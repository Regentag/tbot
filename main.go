package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	toml "github.com/pelletier/go-toml"
	tb "gopkg.in/tucnak/telebot.v2"

	echo "github.com/labstack/echo/v4"
)

func main() {
	log.Println("Telegram bot framework")

	// load config
	cfg, tomlerr := toml.LoadFile("./config.toml")
	if tomlerr != nil {
		log.Fatal("Can't load config.toml")
	}

	// chat id database
	db := ChatDB{}
	dberr := db.Open("./chatdb.db")
	if dberr != nil {
		log.Fatalf("Can't open chatdb.")
		return
	}
	defer db.Close()

	// load configuration
	telegramToken := cfg.Get("bot.token").(string)
	webaddr := cfg.Get("bot.webaddr").(string)

	msgWelcome := cfg.Get("message.welcome").(string)
	msgBye := cfg.Get("message.bye").(string)
	msgOnText := cfg.Get("message.ontext").(string)

	// logging configuration
	log.Println()
	log.Println("Configurations")
	log.Printf("Telegram API Token : %s", telegramToken)
	log.Printf("Web address/port : %s", webaddr)

	// logging saved chat session
	log.Println()
	savedChats, dberr := db.GetChatList()
	if dberr == nil {
		log.Printf("Saved chat session(s) : %d", len(savedChats))
	} else {
		log.Println("Can't read chatdb.")
		log.Fatalf("Error: %s", dberr.Error())
	}

	// create telegram bot
	b, err := tb.NewBot(tb.Settings{
		Token: telegramToken,
		// You can also set custom API URL. If field is empty it equals to "https://api.telegram.org"
		URL:    "",
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/start", func(m *tb.Message) {
		err := db.AddChat(fmt.Sprintf("%d", m.Chat.ID))
		if err != nil {
			log.Printf("Can't add recipient info to chatdb: %d", m.Chat.ID)
			log.Println(err)
			b.Send(m.Chat, "Temporary DB error. Please send `/start' command again.")
		} else {
			log.Printf("New chat session: recipient id %d, type is %v.",
				m.Chat.ID, m.Chat.Type)
			b.Send(m.Chat, msgWelcome)
		}
	})

	b.Handle("/bye", func(m *tb.Message) {
		b.Send(m.Chat, msgBye)
		log.Printf("Chat session finished: recipient id %d, type is %v.",
			m.Chat.ID, m.Chat.Type)
		err := db.DelChat(fmt.Sprintf("%d", m.Chat.ID))
		if err != nil {
			log.Printf("Can't delete recipient from chatdb: %d", m.Chat.ID)
			log.Println(err)
			b.Send(m.Chat, "Temporary DB error. Please send `/bye' command again.")
		}
	})

	b.Handle("/debug", func(m *tb.Message) {
		b.Send(m.Chat, "Chat list")
		chat, err := db.GetChatList()
		if err != nil {
			b.Send(m.Chat, "Query error: "+err.Error())
		} else {
			b.Send(m.Chat, fmt.Sprintf("%d chats.", len(chat)))
		}
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		b.Send(m.Chat, msgOnText)
	})

	// start telegram bot
	go func() {
		b.Start()
	}()

	// start web interface
	e := echo.New()
	handler := func(c echo.Context) error {
		msg := c.FormValue("msg")
		err := SendMessageToAll(b, &db, msg)
		if err == nil {
			return c.String(http.StatusOK, "Ok")
		} else {
			return c.String(http.StatusInternalServerError, "Send error: "+err.Error())
		}
	}

	e.GET("/send", handler)
	e.POST("/send", handler)

	// Start server
	e.Logger.Fatal(e.Start(webaddr))
}

type ChatRecipient struct {
	ChatId string
}

func (c ChatRecipient) Recipient() string {
	return c.ChatId
}

func SendMessageToAll(bot *tb.Bot, db *ChatDB, msg string) error {
	log.Printf("Send message to all recipients:\n%s", msg)

	chats, err := db.GetChatList()

	if err != nil {
		log.Printf("send message: chatdb error. %s", err.Error())
		return err
	}

	log.Printf("Recipients : %d", len(chats))

	// send messages
	for _, chat := range chats {
		_, senderr := bot.Send(ChatRecipient{ChatId: chat}, msg)

		if senderr != nil {
			errcnt, _ := db.GetErrorCount(chat)
			db.SetErrorCount(chat, errcnt+1)

			log.Printf("  Send error. recipient %s, err count = %d", chat, errcnt+1)
		}
	}

	// cleanup  error recipients (errcount > 3)
	errchats, err := db.GetErrorChatList(3)
	if err != nil {
		log.Printf("cleanup error recipients: chatdb error. %s", err.Error())
		return err
	}

	if len(errchats) > 0 {
		for _, errchat := range errchats {
			log.Printf("error recipient %s is removed from chatdb.", errchat)
			db.DelChat(errchat)
		}
	}

	return nil
}
