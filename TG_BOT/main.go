package main

import (
	"log"
	"os"

	"github.com/glebarez/sqlite"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

type DateIdea struct {
	gorm.Model
	Category    string
	Description string
}

var DB *gorm.DB

func initDB() {
	var err error

	DB, err = gorm.Open(sqlite.Open("bot.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}

	err = DB.AutoMigrate(&DateIdea{})
	if err != nil {
		log.Fatalf("Ошибка миграции БД: %v", err)
	}

	log.Println("База данных успешно инициализирована!")

	// Если база данных пустая, наполняем её всеми твоими классными идеями!
	var count int64
	DB.Model(&DateIdea{}).Count(&count)
	if count == 0 {
		initialIdeas := []DateIdea{
			{Category: "На улице 🌳", Description: "Устройте ночной пикник на крыше или в парке с гирляндами на батарейках, термосом с какао и просмотром неонового заката."},
			{Category: "Активный отдых ⚡", Description: "Сходите в современное VR-пространство или на технологичный интерактивный аттракцион, чтобы побегать в виртуальной реальности."},
			{Category: "Дома 🏠", Description: "Устройте кулинарный поединок: выберите случайный рецепт сложного десерта или коктейля, который вы оба никогда не пробовали готовить, и сделайте его вместе под виниловый вайб."},
			{Category: "Культурная программа 🎭", Description: "Посетите выставку современного цифрового искусства (медиа-арт) или галерею с уникальными неоновыми инсталляциями."},
			{Category: "На улице 🌳", Description: "Посетить альпака парк в сокольниках"},
			{Category: "Активный отдых ⚡", Description: "Веломаршрут Зелёное кольцо, либо просто покататься на велосипеде"},
			{Category: "На улице 🌳", Description: "Кинотеатр под открытым небом"},
		}
		DB.Create(&initialIdeas)
		log.Println("База данных была пуста, успешно добавили стартовые идеи!")
	}
}

func main() {
	initDB()

	botToken := os.Getenv("TELEGRAM_APITOKEN")
	if botToken == "" {
		log.Fatal("Переменная окружения TELEGRAM_APITOKEN не установлена!")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	commands := []tgbotapi.BotCommand{
		{
			Command:     "start",
			Description: "Запустить бота и открыть главное меню",
		},
		{
			Command:     "help",
			Description: "Показать справку по командам",
		},
	}

	setCommandsConfig := tgbotapi.NewSetMyCommands(commands...)
	if _, err := bot.Request(setCommandsConfig); err != nil {
		log.Printf("Не удалось установить меню команд: %v", err)
	}
	log.Println("Нижнее меню команд успешно настроено!")

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
			case "menu_ideas", "next_idea":
				var randomIdea DateIdea

				result := DB.Order("RANDOM()").First(&randomIdea)

				if result.Error != nil {
					log.Printf("Ошибка при получении идеи из БД: %v", result.Error)
					responseText = "Ой, не удалось получить идею из базы данных. Попробуйте еще раз! 😢"
				} else {
					responseText = "✨ **Идея для вашего свидания!** ✨\n\n" +
						"**Категория:** " + randomIdea.Category + "\n" +
						"**Что делаем:** " + randomIdea.Description
				}

				btnNext := tgbotapi.NewInlineKeyboardButtonData("👉 Другая идея", "next_idea")
				btnBack := tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад в меню", "go_to_main")

				inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(btnNext, btnBack),
				)

				editMsg := tgbotapi.NewEditMessageTextAndMarkup(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
					responseText,
					inlineKeyboard,
				)

				editMsg.ParseMode = "Markdown"
				bot.Send(editMsg)
				continue // Переходим к следующему апдейту, не отправляя лишний текст

			case "menu_remind":
				responseText = "Тут мы настроим напоминания о годовщинах и днях рождения. Функция в разработке 📅"
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, responseText)
				bot.Send(msg)
				continue

			case "go_to_main":
				responseText = "Привет! Добро пожаловать в Date Romantic Bot 👩‍❤️‍👨\n\nЯ помогу тебе не забыть про важные даты и подкину крутые идеи для свиданий!"

				btnIdeas := tgbotapi.NewInlineKeyboardButtonData("Идеи для свиданий 💡", "menu_ideas")
				btnRemind := tgbotapi.NewInlineKeyboardButtonData("Напомнить о дате 📅", "menu_remind")

				mainKeyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(btnIdeas, btnRemind),
				)

				editMsg := tgbotapi.NewEditMessageTextAndMarkup(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
					responseText,
					mainKeyboard,
				)

				editMsg.ParseMode = "Markdown"

				if _, err := bot.Send(editMsg); err != nil {
					log.Printf("Ошибка при возврате в меню: %v", err)
				}
				continue
			}
		}

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
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Не удалось отправить сообщение: %v", err)
		}
	}
}
