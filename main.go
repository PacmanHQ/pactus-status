package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/kehiy/pactatus/client"
	"github.com/pactus-project/pactus/util"
	pactus "github.com/pactus-project/pactus/www/grpc/gen/go"
)

var rpcNodes = []string{"181.214.208.165:50051", "bootstrap1.pactus.org:50051", "bootstrap2.pactus.org:50051", "bootstrap3.pactus.org:50051", "bootstrap4.pactus.org:50051", "151.115.110.114:50051", "188.121.116.247:50051"}

func main() {
	ctx := context.Background()

	fmt.Println("starting")

	cmgr := client.NewClientMgr(ctx)

	for _, rn := range rpcNodes {
		c, e := client.NewClient(rn)
		if e != nil {
			fmt.Printf("error: %v adding client %s\n", e, rn)
			continue
		}
		cmgr.AddClient(c)
		fmt.Printf("client added %s\n", rn)
	}

	botToken := os.Args[1]
	b, err := bot.New(botToken, bot.WithAllowedUpdates(bot.AllowedUpdates{}))
	if err != nil {
		panic(err)
	}

	go PostUpdates(ctx, b, cmgr)

	b.Start(ctx)
}

func PostUpdates(ctx context.Context, b *bot.Bot, cmgr *client.Mgr) {
	m, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: "@pactus_status",
		Text: ".",
	})

	messageID := m.ID

	for {
		fmt.Println("posting new update!")
		status, lbt, lbh, td := networkHealth(cmgr)
		bi, err := cmgr.GetBlockchainInfo()
		if err != nil {
			panic(err)
		}

		fmt.Println("got network health and Blockcahin info successfully")

		cs, err := cmgr.GetCirculatingSupply()
		if err != nil {
			panic(err)
		}

		fmt.Println("got circ supply successfully")

		msg := makeMessage(bi, cs, td, status, lbt, lbh)
		_, err = b.EditMessageText(ctx, makeMessageParams(msg, messageID))
		if err != nil {
			fmt.Printf("can't post updates: %v\n", err)
		}
		fmt.Println("updated posted successfully")

		time.Sleep(7 * time.Second)
	}
}

func makeMessage(b *pactus.GetBlockchainInfoResponse, c, timeDiff int64, status, lastBlkTime string, lastBlkH uint32) string {
	var s strings.Builder

	s.WriteString("🔴 Pactus Network Status Update\n\n")
	s.WriteString("ℹ️ Blockchain Info\n")
	s.WriteString(fmt.Sprintf("⛓️ **%s** is Last Block Height\n", formatNumber(int64(lastBlkH))))
	s.WriteString(fmt.Sprintf("👤 **%v** Active Accounts\n", formatNumber(int64(b.TotalAccounts))))
	s.WriteString(fmt.Sprintf("🕵️ **%v** Total Validators\n", formatNumber(int64(b.TotalValidators))))
	s.WriteString(fmt.Sprintf("🦾 **%v** Total PAC Staked\n", formatNumber(int64(util.ChangeToCoin(b.TotalPower)))))
	s.WriteString(fmt.Sprintf("🦾 **%v PAC** is Committee Power\n", formatNumber(int64(util.ChangeToCoin(b.CommitteePower)))))
	s.WriteString(fmt.Sprintf("🔄 **%v PAC** is in Circulating\n\n", formatNumber(int64(util.ChangeToCoin(c)))))

	s.WriteString("🧑🏻‍⚕️ Network Status\n\n")
	s.WriteString(fmt.Sprintf("```Details Network is %s\n\n%s is The LastBlock time and there is %v seconds passed from last block```", status, lastBlkTime, timeDiff))

	return s.String()
}

func networkHealth(cmgr *client.Mgr) (string, string, uint32, int64) {
	lastBlockTime, lastBlockHeight := cmgr.GetLastBlockTime()
	lastBlockTimeFormatted := time.Unix(int64(lastBlockTime), 0).Format("02/01/2006, 15:04:05")
	currentTime := time.Now()

	timeDiff := (currentTime.Unix() - int64(lastBlockTime))

	healthStatus := true
	if timeDiff > 15 {
		healthStatus = false
	}

	var status string
	if healthStatus {
		status = "Healthy✅"
	} else {
		status = "UnHealthy❌"
	}

	return status, lastBlockTimeFormatted, lastBlockHeight, timeDiff
}

func makeMessageParams(t string, mi int) *bot.EditMessageTextParams {
	return &bot.EditMessageTextParams{
		ChatID:    "@pactus_status",
		Text:      t,
		ParseMode: models.ParseModeMarkdown,
		MessageID: mi,
	}
}

func formatNumber(num int64) string {
	numStr := strconv.FormatInt(num, 10)

	var formattedNum string
	for i, c := range numStr {
		if (i > 0) && (len(numStr)-i)%3 == 0 {
			formattedNum += ","
		}
		formattedNum += string(c)
	}

	return formattedNum
}
