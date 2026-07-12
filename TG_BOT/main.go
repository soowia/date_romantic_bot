package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	botToken := os.Getenv("TELEGRAM_APITOKEN")
	if botToken == "" {
		log.Fatal("Переменная окружения TELEGRAM_APITOKEN не установлена!")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Авторизовались под аккаунтом %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {

			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
			if _, err := bot.Request(callback); err != nil {
				log.Printf("Не удалось ответить на callback: %v", err)
			}

			var responseText string
			switch update.CallbackQuery.Data {
			case "menu_ideas":
				responseText = "Здесь скоро будет генератор крутых идей для свиданий! Напиши, что бы вам хотелось: кино, ресторан или экстрим? 😉"
			case "menu_remind":
				responseText = "Тут мы настроим напоминания о годовщинах и днях рождения. Функция в разработке 📅"
			}

			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, responseText)
			bot.Send(msg)
			continue
		}
		// -----------------------------------------------------
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] написал: %s", update.Message.From.UserName, update.Message.Text)

		if update.Message.IsCommand() {
			var replyText string

			switch update.Message.Command() {
			case "start":
				replyText = "Привет! Добро пожаловать в Date Romantic Bot 👩‍❤️‍👨\n\nЯ помогу тебе не забыть про важные даты и подкину крутые идеи для свиданий!"
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)

				btnIdeas := tgbotapi.NewInlineKeyboardButtonData("Идеи для свиданий 💡", "menu_ideas")
				btnRemind := tgbotapi.NewInlineKeyboardButtonData("Напомнить о дате 📅", "menu_remind")

				numericKeyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(btnIdeas, btnRemind),
				)

				msg.ReplyMarkup = numericKeyboard

				bot.Send(msg)
				continue
			case "help":
				replyText = "Доступные команды:\n/start - запустить бота\n/help - показать это меню"
			default:
				replyText = "Я не знаю такую команду 🤷‍♂️"
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
			bot.Send(msg)
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ты написал обычный текст: "+update.Message.Text)
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Не удалось отправить сообщение: %v", err)
		}
	}
}
