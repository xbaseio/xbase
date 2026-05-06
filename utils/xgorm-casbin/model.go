package xcasbin

const (
	defaultTableName     = "casbin_policy"
	dropPolicyTableSql   = `DROP TABLE IF EXISTS %s`
	createPolicyTableSql = `
CREATE TABLE IF NOT EXISTS %s (
	ptype VARCHAR(10) NOT NULL DEFAULT '',
	v0 VARCHAR(256) NOT NULL DEFAULT '',
	v1 VARCHAR(256) NOT NULL DEFAULT '',
	v2 VARCHAR(256) NOT NULL DEFAULT '',
	v3 VARCHAR(256) NOT NULL DEFAULT '',
	v4 VARCHAR(256) NOT NULL DEFAULT '',
	v5 VARCHAR(256) NOT NULL DEFAULT ''
) COMMENT = 'policy table';
`
)

var policyColumns = defaultPolicyColumns{
	PType: "ptype",
	V0:    "v0",
	V1:    "v1",
	V2:    "v2",
	V3:    "v3",
	V4:    "v4",
	V5:    "v5",
}

type defaultPolicyColumns struct {
	PType string // ptype
	V0    string // V0
	V1    string // V1
	V2    string // V2
	V3    string // V3
	V4    string // V4
	V5    string // V5
}

// policy rule entity
type policyRule struct {
	PType string `gorm:"column:ptype" json:"ptype"`
	V0    string `gorm:"column:v0" json:"v0"`
	V1    string `gorm:"column:v1" json:"v1"`
	V2    string `gorm:"column:v2" json:"v2"`
	V3    string `gorm:"column:v3" json:"v3"`
	V4    string `gorm:"column:v4" json:"v4"`
	V5    string `gorm:"column:v5" json:"v5"`
}
