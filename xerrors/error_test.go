package xerrors_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xbaseio/xbase/codes"
	"github.com/xbaseio/xbase/xerrors"
)

type PayChannelChild struct {
	ID          uint      `gorm:"primarykey"`
	ThirdName   string    `gorm:"column:third_name;type:varchar(191);not null;comment:第三方支付名称;uniqueIndex:uk_third_name_code,priority:1"`
	Code        string    `gorm:"column:code;type:varchar(191);not null;comment:子渠道code;uniqueIndex:uk_third_name_code,priority:2"`
	Name        string    `gorm:"column:name;type:varchar(191);not null;comment:子渠道名称"`
	ChannelName string    `gorm:"column:channel_name;type:varchar(191);comment:所属渠道ID"`
	MinAmount   float64   `gorm:"column:min_amount;type:decimal(10,2);not null;comment:最小金额"`
	MaxAmount   float64   `gorm:"column:max_amount;type:decimal(10,2);not null;comment:最大金额"`
	Status      int       `gorm:"column:status;not null;comment:状态0关闭1开启"`
	Fee         float64   `gorm:"column:fee;type:decimal(5,2);not null;comment:手续费小数表示百分比"`
	SortID      int64     `gorm:"column:sort_id;default:0;comment:排序ID"`
	Operator    string    `gorm:"column:operator;type:varchar(191);comment:操作人"`
	OperatorID  int64     `gorm:"column:operator_id;comment:操作人ID"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at" form:"created_at" sql:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at" form:"updated_at" sql:"updated_at"`
	UseCount    int64     `gorm:"-"`
}

func TestNew(t *testing.T) {
	innerErr := xerrors.NewCode(
		codes.NewCode(2, "core error"),
		"aaaaaaa",
	)

	err := xerrors.NewCode(
		//"not found",
		codes.NewCode(1, "not found"),
		innerErr.Error(),
	)

	t.Log(err)
	t.Log(err.Code())
	t.Log(err.Next())
	t.Log(err.Cause())
	fmt.Println(fmt.Sprintf("%+v", err))
}

var (
	cfg atomic.Pointer[map[string][]PayChannelChild]
)

func Test_Pointer(t *testing.T) {

	cfg.Store(&map[string][]PayChannelChild{
		"test": {
			{
				ID:          1,
				ThirdName:   "test",
				Code:        "test",
				Name:        "test",
				ChannelName: "test",
				MinAmount:   0.01,
				MaxAmount:   10000,
				Status:      1,
				Fee:         0.01,
				SortID:      1,
				Operator:    "test",
				OperatorID:  1,
			},
		}})

	child := findByChannelName("test", 100)

	child.UseCount += 10
	t.Log(child)

	child2 := findByChannelName("test", 100)
	t.Log(child2)
}
func findByChannelName(channelName string, amount float64) *PayChannelChild {
	cfgMap := cfg.Load()
	if cfgMap == nil {
		return nil
	}
	children, ok := (*cfgMap)[channelName]
	if !ok {
		return nil
	}
	return getPayChannelChild(children, amount)
}
func getPayChannelChild(children []PayChannelChild, amount float64) *PayChannelChild {
	var (
		getchild    *PayChannelChild
		minUseCount = int64(0)
	)

	for i := range children {
		child := &children[i]

		if amount >= child.MinAmount-0.5 && amount <= child.MaxAmount+0.5 {
			if getchild == nil || child.UseCount < minUseCount {
				minUseCount = child.UseCount
				getchild = child
			}
		}
	}

	return getchild
}
