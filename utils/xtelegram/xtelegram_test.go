package xtelegram_test

import (
	"fmt"
	"testing"

	"github.com/xbaseio/xbase/config"
	"github.com/xbaseio/xbase/config/file"
	"github.com/xbaseio/xbase/log"

	//optionChannelDao "github.com/xbaseio/xbase/utils/dao/option-channel"
	//optionListenerAddressDao "github.com/xbaseio/xbase/utils/dao/option-listener-address"
	//optiontelegramcmd "github.com/xbaseio/xbase/utils/option/option-telegram-cmd"

	"github.com/xbaseio/xbase/utils/xtelegram"
	tgtypes "github.com/xbaseio/xbase/utils/xtelegram/tg-types"
)

const (
	bot_token = "6867997452:AAFYZXHAC_TDvcfBiYto2ShutRSiUcboa04"
)
const (
	OfficialChannelCode = ""
	TRON                = ""
)

func init() {
	// 设置配置中心
	config.SetConfigurator(config.NewConfigurator(config.WithSources(file.NewSource())))
}
func TestClient_SendMessage(t *testing.T) {

	msg := `🎉 *Congratulations,${winner_name}!* 🎉
	You won with a *${multiplier}x* multiplier!
		This will earn you a total of *${win_amount}${currency}*.
	You're on fire! 🔥 Keep it up, aim for the moon!`
	replaces := make(map[string]string)

	replaces["winner_name"] = "黄老师"
	replaces["win_amount"] = fmt.Sprintf("%.4f", 1.0)
	replaces["multiplier"] = fmt.Sprintf("%.4f", 1.1)
	replaces["game_name"] = "黄老师"
	replaces["currency"] = "csd"

	xMsg, err := xtelegram.NewXTelegramMessage(bot_token,
		xtelegram.WithText(msg),
		xtelegram.WithDebug(true),
		xtelegram.WithMessageThreadID(-1002066585210),
		xtelegram.WithMsgType(tgtypes.RobotMsgTypePhoto),
		xtelegram.WithParseMode(tgtypes.ModeMarkdown))
	if err == nil {
		log.Errorf("%v", err)
		return
	}
	xMsg.SendMessage(33, replaces)

}

func TestClient_SendMessageCmd(t *testing.T) {
	/*
	   ctx := context.Background()
	   channelCfg, err := optionChannelDao.Instance().GetChannel(ctx, OfficialChannelCode)

	   	if err != nil {
	   		log.Errorf("%v", err)
	   		return
	   	}

	   	if channelCfg == nil {
	   		log.Errorf("channelCfg is nil")
	   		return
	   	}

	   trc20Address := optionListenerAddressDao.Instance().GetAddressByChannelCode(OfficialChannelCode, TRON)

	   	if trc20Address == "" {
	   		log.Errorf("trc20Address is nil :%v", OfficialChannelCode)
	   		return
	   	}

	   cmdMsg := optiontelegramcmd.GetChanCodeCmd(OfficialChannelCode, tgtypes.XTelegramCmd_Button_RechargeOtherAddresses)

	   	if cmdMsg == nil {
	   		return
	   	}

	   xMsg, err := tgmsg.NewXTelegramMessage(channelCfg.TelegramCfg.MainRobotToken,

	   	tgmsg.WithText(cmdMsg.Text),
	   	tgmsg.WithDebug(true),
	   	tgmsg.WithCmd(cmdMsg.Cmd),
	   	tgmsg.WithMsgType(cmdMsg.Type),
	   	tgmsg.WithParseMode(cmdMsg.ParseMode))

	   	if err != nil {
	   		return
	   	}

	   	if xMsg == nil {
	   		return
	   	}

	   orderID := xstr.SerialNO()

	   	if err := xMsg.SetExtraCallBackData(orderID); err != nil {
	   		log.Warnf("%v", err)
	   		return
	   	}

	   	expandMap := map[string]string{
	   		tgtemplate.ComboKindEnergyFlashRentalNumKey:  xconv.String(channelCfg.ChannelCfg.ComboKindEnergyFlashRental.Duration),
	   		tgtemplate.ComboKindEnergyFlashRentalNameKey: channelCfg.ChannelCfg.ComboKindEnergyFlashRental.ComboKind.Name(),
	   		tgtemplate.PriceNumKey:                       "1",
	   		tgtemplate.NotActivatedAddressCountKey:       "2",
	   		tgtemplate.ReceivingAddressCountKey:          "3",
	   		tgtemplate.EnergyFeeKey:                      "4",
	   		tgtemplate.ActivationfeeKey:                  "5",
	   		tgtemplate.PayAmountKey:                      "6",
	   		tgtemplate.Tron20AddressKey:                  trc20Address,
	   	}

	   	if _, err := xMsg.SendMessage(7026994919, expandMap); err != nil {
	   		log.Warnf("sendMessage:%v", err)
	   	}
	*/
}
