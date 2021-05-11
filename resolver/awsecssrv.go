package resolver

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/thisissc/awsclient"
	"github.com/thisissc/config"
	"github.com/thisissc/radixclient"
	gr "google.golang.org/grpc/resolver"
)

func init() {
	config.SetConfigFile("config.toml")
	config.LoadConfig("AWS", &awsclient.Config{})
	config.LoadConfig("Radix", &radixclient.Config{})
	gr.Register(&awsEndpointsBuilder{})
}

const (
	cacheKey = "ZA:AwsEcsServiceEndpoint:%s"
	cacheTTL = 5
)

type awsEndpointsBuilder struct{}

func (*awsEndpointsBuilder) Build(target gr.Target, cc gr.ClientConn, opts gr.BuildOptions) (gr.Resolver, error) {
	a := &awsEndpointsResolver{
		session: awsclient.GetSession(),
		target:  target,
		cc:      cc,
	}
	go a.start()
	return a, nil
}

func (*awsEndpointsBuilder) Scheme() string {
	return "awsecssrv"
}

type awsEndpointsResolver struct {
	session     *session.Session
	target      gr.Target
	cc          gr.ClientConn
	serviceName string
	stopSignal  chan struct{}
	addrs       []string
}

func (a *awsEndpointsResolver) ResolveNow(option gr.ResolveNowOptions) {}

func (a *awsEndpointsResolver) Close() {
	a.stopSignal <- struct{}{}
}

func (a *awsEndpointsResolver) start() {
	a.serviceName = a.target.Endpoint
	a.stopSignal = make(chan struct{}, 1)

	// 1/2 ttl更新一次地址列表，防止缓存击穿
	t := time.NewTicker(cacheTTL / 2 * time.Second)
	defer t.Stop()
	for {
		a.getAddr()
		<-t.C
	}
}

func (a *awsEndpointsResolver) updateState(addrs []string) {
	a.addrs = addrs

	addressList := make([]gr.Address, len(addrs))
	for i, a := range addrs {
		addressList[i] = gr.Address{Addr: a}
	}
	a.cc.UpdateState(gr.State{Addresses: addressList})
}

func (a *awsEndpointsResolver) getAddr() {
	key := fmt.Sprintf(cacheKey, a.serviceName)

	var addrs []string
	err := radixclient.LoadFromRedis(key, &addrs)
	if err != nil {
		addrs = awsclient.GetEcsServiceAddrList(a.session, a.target.Authority, a.target.Endpoint)
		err := radixclient.Save2RedisMutex(key, cacheTTL, addrs)
		if err != nil {
			log.Println(err)
		}
	}

	//addrs = []string{"127.0.0.1:8080"} // 测试
	if !a.compareAddrs(addrs) {
		// 有变化时更新
		a.updateState(addrs)
	}
}

func (a *awsEndpointsResolver) compareAddrs(addrs []string) bool {
	l1, l2 := len(a.addrs), len(addrs)
	if l1 != l2 {
		return false
	} else if l1 == 1 {
		return a.addrs[0] == addrs[0]
	}

	temp := make(map[string]struct{}, l1)
	for _, addr := range a.addrs {
		temp[addr] = struct{}{}
	}

	for _, addr := range addrs {
		if _, ok := temp[addr]; !ok {
			return false
		}
	}

	return true
}
