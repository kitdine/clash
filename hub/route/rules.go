package route

import (
	"context"
	"net/http"

	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/tunnel"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

func ruleRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", getRules)
	r.Put("/", getRulesets)
	return r
}

type Rule struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
}

type RuleSet struct {
	Rule
	LastUpdate string `json:"last-update"`
}

type Rules interface {}

func getRules(w http.ResponseWriter, r *http.Request) {
	rawRules := tunnel.Rules()

	rules := []Rules{}
	for _, rule := range rawRules {
		r := Rule{
			Type:    rule.RuleType().String(),
			Payload: rule.Payload(),
			Proxy:   rule.Adapter(),
		}
		if mr, ok := rule.(C.RuleSet); ok {
			rules = append(rules, RuleSet{
				Rule:       r,
				LastUpdate: mr.LastUpdate(),
			})
		} else {
			rules = append(rules, r)
		}
	}

	render.JSON(w, r, render.M{
		"rules": rules,
	})
}

func getRulesets(w http.ResponseWriter, r *http.Request) {
	rawRules := tunnel.Rules()

	success := make(chan C.RuleSet)
	ctx, cancel := context.WithTimeout(context.Background(), C.DownloadTimeout)
	count := 0
	defer cancel()
	for _, rule := range rawRules {
		if rule.RuleType() == C.Ruleset {
			if r, ok := rule.(C.RuleSet); ok {
				count++
				go r.Update(ctx, success)
			}
		}
	}
	rules := []Rules{}
	if count == 0 {
		render.JSON(w, r, render.M{
			"rules": rules,
		})
		return
	}
	for {
		select {
		case rule := <-success:
			rules = append(rules, RuleSet{
				Rule: Rule{
					Type:    rule.RuleType().String(),
					Payload: rule.Payload(),
					Proxy:   rule.Adapter(),
				},
				LastUpdate: rule.LastUpdate(),
			})
			if count == len(rules) {
				render.JSON(w, r, render.M{
					"rules": rules,
				})
				return
			}
		case <-ctx.Done():
			render.JSON(w, r, render.M{
				"rules": rules,
			})
			return
		}
	}
}
