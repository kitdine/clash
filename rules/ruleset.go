package rules

import (
	"fmt"
	"github.com/Dreamacro/clash/adapters/provider"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/log"
	"strings"
)

type Ruleset struct {
	ruleProvider provider.RuleProvider
	adapter      string
}

func (r *Ruleset) RuleType() C.RuleType {
	return C.Ruleset
}

func (r *Ruleset) Adapter() string {
	return r.adapter
}

func (r *Ruleset) Payload() string {
	return r.ruleProvider.Name()
}

func (r *Ruleset) NoResolveIP() bool {
	return true
}

func (r *Ruleset) Match(metadata *C.Metadata) bool {
	for _, rule := range r.ruleProvider.Rules() {
		if rule.Match(metadata) {
			return true
		}
	}
	return false
}

func NewRuleset(name string, mapping map[string]interface{}, adapter string) (*Ruleset, provider.RuleProvider, error) {
	provider, err := provider.ParseRuleProvider(name, mapping, parseRules, adapter)
	if err != nil {
		return nil, nil, err
	}
	log.Infoln("Start initial provider %s", provider.Name())
	if err := provider.Initial(); err != nil {
		return nil, nil, err
	}
	return &Ruleset{
		ruleProvider: provider,
		adapter:      adapter,
	}, provider, nil
}

func parseRules(name string, elm interface{}, target string) []C.Rule {
	rules := []C.Rule{}

	rulesConfig := elm.([]string)
	// parse rules
	for idx, line := range rulesConfig {
		rule := trimArr(strings.Split(line, ","))
		var (
			payload string
			params  = []string{}
		)

		switch l := len(rule); {
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
			log.Errorln("Rule-Set[%s] Rules[%d] [%s] error: %s", name, idx, line, parseErr.Error())
			continue
		}

		rules = append(rules, parsed)
	}
	return rules
}

func trimArr(arr []string) (r []string) {
	for _, e := range arr {
		r = append(r, strings.Trim(e, " "))
	}
	return
}
