package rules

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/log"
)

type Ruleset struct {
	url         string
	rules       []C.Rule
	adapter     string
	isNoResolve bool
	lastUpdate  time.Time
	done        chan struct{}
}

func (r *Ruleset) RuleType() C.RuleType {
	return C.Ruleset
}

func (r *Ruleset) Match(metadata *C.Metadata) bool {
	for _, rule := range r.rules {
		if rule.Match(metadata) {
			return true
		}
	}
	return false
}

func (r *Ruleset) Adapter() string {
	return r.adapter
}

func (r *Ruleset) Payload() string {
	return r.url
}

func (r *Ruleset) NoResolveIP() bool {
	return r.isNoResolve
}

func (r *Ruleset) Destroy() {
	r.done <- struct{}{}
}

func (r *Ruleset) loop() {
	tick := time.NewTicker(C.UpdateInterval)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go r.Update(ctx, make(chan C.RuleSet))
Loop:
	for {
		select {
		case <-tick.C:
			go r.Update(ctx, make(chan C.RuleSet))
		case <-r.done:
			break Loop
		}
	}
}

func trimArr(arr []string) (r []string) {
	for _, e := range arr {
		r = append(r, strings.Trim(e, " "))
	}
	return
}

func (r *Ruleset) Update(ctx context.Context, success chan C.RuleSet) {
	isNoResolve := true

	rules := []C.Rule{}

	req, err := http.NewRequestWithContext(ctx, "GET", r.url, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	rawRules, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		log.Errorln(err.Error())
		return
	}

	rulesConfig := strings.Split(string(rawRules), "\n")
	// parse rules
	for _, line := range rulesConfig {
		rule := trimArr(strings.Split(line, ","))
		var (
			payload string
			target  string
			params  = []string{}
		)

		target = r.adapter

		switch l := len(rule); {
		case l == 1:
		case l == 2:
			payload = rule[1]
		case l >= 3:
			payload = rule[1]
			params = rule[2:]
		default:
			continue
		}

		rule = trimArr(rule)
		params = trimArr(params)
		var (
			parseErr error
			parsed   C.Rule
		)

		switch rule[0] {
		case "DOMAIN":
			parsed = NewDomain(payload, target)
		case "DOMAIN-SUFFIX":
			parsed = NewDomainSuffix(payload, target)
		case "DOMAIN-KEYWORD":
			parsed = NewDomainKeyword(payload, target)
		case "GEOIP":
			noResolve := HasNoResolve(params)
			parsed = NewGEOIP(payload, target, noResolve)
		case "IP-CIDR", "IP-CIDR6":
			noResolve := HasNoResolve(params)
			parsed, parseErr = NewIPCIDR(payload, target, WithIPCIDRNoResolve(noResolve))
		// deprecated when bump to 1.0
		case "SOURCE-IP-CIDR":
			fallthrough
		case "SRC-IP-CIDR":
			parsed, parseErr = NewIPCIDR(payload, target, WithIPCIDRSourceIP(true), WithIPCIDRNoResolve(true))
		case "SRC-PORT":
			parsed, parseErr = NewPort(payload, target, true)
		case "DST-PORT":
			parsed, parseErr = NewPort(payload, target, false)
		case "MATCH":
			fallthrough
		// deprecated when bump to 1.0
		case "FINAL":
			parsed = NewMatch(target)
		default:
			parseErr = fmt.Errorf("unsupported rule type %s", rule[0])
		}

		if parseErr != nil {
			continue
		}

		rules = append(rules, parsed)
	}

	r.isNoResolve = isNoResolve
	r.rules = rules
	r.lastUpdate = time.Now()

	success <- r
}

func (r *Ruleset) LastUpdate() string {
	return r.lastUpdate.Format(time.UnixDate)
}

func NewRuleset(url string, adapter string) *Ruleset {
	rs := Ruleset{
		url:     url,
		adapter: adapter,
		done:    make(chan struct{}),
	}
	go rs.loop()
	return &rs
}
