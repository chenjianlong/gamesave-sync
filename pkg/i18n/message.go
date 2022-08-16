package i18n

import (
	"encoding/json"
	. "github.com/chenjianlong/gamesave-sync/pkg/gsutils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"log"
)

var bundle *i18n.Bundle
var loc *i18n.Localizer

func InitBundle(locale string) {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	bundle.MustLoadMessageFile("i18n/en.json")
	bundle.MustLoadMessageFile("i18n/zh-CN.json")
	loc = i18n.NewLocalizer(bundle, locale)
}
func GetSyncGameMessage(msgID string) string {
	name, _, _ := loc.LocalizeWithTag(&i18n.LocalizeConfig{
		MessageID: msgID,
	})

	if name == "" {
		name = msgID
	}

	msg, _, err := loc.LocalizeWithTag(&i18n.LocalizeConfig{
		MessageID: "SyncGame",
		TemplateData: map[string]interface{}{
			"Name": name,
		},
	})
	if msg == "" {
		CheckError(err)
		log.Fatal("Message with SyncGame ID not found")
	}
	return msg
}
