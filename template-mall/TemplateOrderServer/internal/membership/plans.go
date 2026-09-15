package membership

import "fmt"

const (
	PlanMonth   = "month"
	PlanQuarter = "quarter"
	PlanYear    = "year"

	TemplatePrefix = "membership_"
)

type Plan struct {
	ID       string
	Name     string
	PriceFen int64
}

var plans = map[string]Plan{
	PlanMonth:   {ID: PlanMonth, Name: "包月会员", PriceFen: 1500},
	PlanQuarter: {ID: PlanQuarter, Name: "包季会员", PriceFen: 4000},
	PlanYear:    {ID: PlanYear, Name: "包年会员", PriceFen: 13800},
}

func GetPlan(plan string) (Plan, error) {
	p, ok := plans[plan]
	if !ok {
		return Plan{}, fmt.Errorf("invalid plan")
	}
	return p, nil
}

func TemplateID(plan string) string {
	return TemplatePrefix + plan
}

func PlanFromTemplateID(templateID string) (Plan, bool) {
	if len(templateID) <= len(TemplatePrefix) || templateID[:len(TemplatePrefix)] != TemplatePrefix {
		return Plan{}, false
	}
	p, err := GetPlan(templateID[len(TemplatePrefix):])
	if err != nil {
		return Plan{}, false
	}
	return p, true
}

func IsSubscriptionTemplate(templateID string) bool {
	_, ok := PlanFromTemplateID(templateID)
	return ok
}

func DisplayName(templateID string) string {
	if p, ok := PlanFromTemplateID(templateID); ok {
		return p.Name
	}
	return ""
}
