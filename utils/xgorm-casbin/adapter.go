package xcasbin

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"gorm.io/gorm"
)

type adapter struct {
	db    *gorm.DB
	table string
}

func newAdapter(database interface{}, table string, logger ...Logger) (*adapter, error) {
	var adp *adapter

	switch v := database.(type) {
	case string:
		db, err := newDatabase(v, logger...)
		if err != nil {
			return nil, err
		}

		adp = &adapter{db: db, table: table}
	case *gorm.DB:
		adp = &adapter{db: v, table: table}
	default:
		return nil, errors.New("invalid database")
	}

	if adp.table == "" {
		adp.table = defaultTableName
	}

	err := adp.createPolicyTable()
	if err != nil {
		return nil, err
	}

	return adp, nil
}

func (a *adapter) model() *gorm.DB {
	return a.db.Table(a.table)
}

// create a policy table when it's not exists.
func (a *adapter) createPolicyTable() error {
	return a.db.Exec(fmt.Sprintf(createPolicyTableSql, a.table)).Error
}

// drop policy table from the storage.
func (a *adapter) dropPolicyTable() error {
	return a.db.Exec(fmt.Sprintf(dropPolicyTableSql, a.table)).Error
}

// LoadPolicy loads all policy rules from the storage.
func (a *adapter) LoadPolicy(model model.Model) error {
	var rules []*policyRule

	err := a.model().Scan(&rules).Error
	if err != nil {
		return err
	}

	for _, rule := range rules {
		a.loadPolicyRule(rule, model)
	}

	return nil
}

// SavePolicy Saves all policy rules to the storage.
func (a *adapter) SavePolicy(model model.Model) error {
	err := a.dropPolicyTable()
	if err != nil {
		return err
	}

	err = a.createPolicyTable()
	if err != nil {
		return err
	}

	policyRules := make([]*policyRule, 0)
	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			policyRules = append(policyRules, a.buildPolicyRule(ptype, rule))
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			policyRules = append(policyRules, a.buildPolicyRule(ptype, rule))
		}
	}

	if count := len(policyRules); count > 0 {
		err = a.model().CreateInBatches(policyRules, count).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// AddPolicy adds a policy rule to the storage.
func (a *adapter) AddPolicy(sec string, ptype string, rule []string) error {
	return a.model().Create(a.buildPolicyRule(ptype, rule)).Error
}

// AddPolicies adds policy rules to the storage.
func (a *adapter) AddPolicies(sec string, ptype string, rules [][]string) error {
	if len(rules) == 0 {
		return nil
	}

	policyRules := make([]*policyRule, 0, len(rules))
	for _, rule := range rules {
		policyRules = append(policyRules, a.buildPolicyRule(ptype, rule))
	}

	return a.model().CreateInBatches(policyRules, len(policyRules)).Error
}

// RemovePolicy removes a policy rule from the storage.
func (a *adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return a.deletePolicyRule(a.buildPolicyRule(ptype, rule))
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	rule := &policyRule{PType: ptype}

	if fieldIndex <= 0 && 0 < fieldIndex+len(fieldValues) {
		rule.V0 = fieldValues[0-fieldIndex]
	}

	if fieldIndex <= 1 && 1 < fieldIndex+len(fieldValues) {
		rule.V1 = fieldValues[1-fieldIndex]
	}

	if fieldIndex <= 2 && 2 < fieldIndex+len(fieldValues) {
		rule.V2 = fieldValues[2-fieldIndex]
	}

	if fieldIndex <= 3 && 3 < fieldIndex+len(fieldValues) {
		rule.V3 = fieldValues[3-fieldIndex]
	}

	if fieldIndex <= 4 && 4 < fieldIndex+len(fieldValues) {
		rule.V4 = fieldValues[4-fieldIndex]
	}

	if fieldIndex <= 5 && 5 < fieldIndex+len(fieldValues) {
		rule.V5 = fieldValues[5-fieldIndex]
	}

	return a.deletePolicyRule(rule)
}

// RemovePolicies removes policy rules from the storage (implements the persist.BatchAdapter interface).
func (a *adapter) RemovePolicies(sec string, ptype string, rules [][]string) (err error) {
	db := a.model()

	for _, rule := range rules {
		query := make([]string, 0, 7)
		args := make([]interface{}, 0, 7)
		query = append(query, fmt.Sprintf("%s = ?", policyColumns.PType))
		args = append(args, ptype)

		for i := 0; i <= 5; i++ {
			if len(rule) > i {
				query = append(query, fmt.Sprintf("%s = ?", fmt.Sprintf("v%d", i)))
				args = append(args, rule[i])
			}
		}

		if len(query) > 0 {
			db = db.Or(fmt.Sprintf("(%s)", strings.Join(query, " AND ")), args...)
		}
	}

	return db.Delete(&policyRule{}).Error
}

// UpdatePolicy updates a policy rule from storage.
func (a *adapter) UpdatePolicy(sec string, ptype string, oldRule, newRule []string) error {
	return a.model().Where(a.buildPolicyRule(ptype, oldRule)).Updates(a.buildPolicyRule(ptype, newRule)).Error
}

// UpdatePolicies updates some policy rules to storage, like db, redis.
func (a *adapter) UpdatePolicies(sec string, ptype string, oldRules, newRules [][]string) error {
	if len(oldRules) == 0 || len(newRules) == 0 {
		return nil
	}

	return a.db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < int(math.Min(float64(len(oldRules)), float64(len(newRules)))); i++ {
			err := tx.Table(a.table).Where(a.buildPolicyRule(ptype, newRules[i])).Updates(a.buildPolicyRule(ptype, oldRules[i])).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// load a policy rule
func (a *adapter) loadPolicyRule(rule *policyRule, model model.Model) {
	ruleText := rule.PType

	if rule.V0 != "" {
		ruleText += ", " + rule.V0
	}

	if rule.V1 != "" {
		ruleText += ", " + rule.V1
	}

	if rule.V2 != "" {
		ruleText += ", " + rule.V2
	}

	if rule.V3 != "" {
		ruleText += ", " + rule.V3
	}

	if rule.V4 != "" {
		ruleText += ", " + rule.V4
	}

	if rule.V5 != "" {
		ruleText += ", " + rule.V5
	}

	persist.LoadPolicyLine(ruleText, model)
}

// build a policy rule.
func (a *adapter) buildPolicyRule(ptype string, data []string) *policyRule {
	rule := &policyRule{PType: ptype}

	if len(data) > 0 {
		rule.V0 = data[0]
	}

	if len(data) > 1 {
		rule.V1 = data[1]
	}

	if len(data) > 2 {
		rule.V2 = data[2]
	}

	if len(data) > 3 {
		rule.V3 = data[3]
	}

	if len(data) > 4 {
		rule.V4 = data[4]
	}

	if len(data) > 5 {
		rule.V5 = data[5]
	}

	return rule
}

// deletes a policy rule.
func (a *adapter) deletePolicyRule(rule *policyRule) error {
	where := make(map[string]interface{}, 7)
	where[policyColumns.PType] = rule.PType

	if rule.V0 != "" {
		where[policyColumns.V0] = rule.V0
	}

	if rule.V1 != "" {
		where[policyColumns.V1] = rule.V1
	}

	if rule.V2 != "" {
		where[policyColumns.V2] = rule.V2
	}

	if rule.V3 != "" {
		where[policyColumns.V3] = rule.V3
	}

	if rule.V4 != "" {
		where[policyColumns.V4] = rule.V4
	}

	if rule.V5 != "" {
		where[policyColumns.V5] = rule.V5
	}

	return a.model().Where(where).Delete(&policyRule{}).Error
}
