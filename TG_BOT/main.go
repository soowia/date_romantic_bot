package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/glebarez/sqlite"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

const adminHTML = `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Панель управления Date Bot</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 font-sans leading-normal tracking-normal">
    <div class="container mx-auto px-4 py-8 max-w-2xl">
        <h1 class="text-3xl font-bold text-gray-800 mb-6 text-center">💖 Админка Date Bot</h1>
        
        <!-- Форма добавления -->
        <div class="bg-white p-6 rounded-lg shadow-md mb-8">
            <h2 class="text-xl font-semibold text-gray-700 mb-4">Добавить новую идею</h2>
            <form action="/admin/add" method="POST" class="space-y-4">
                <div>
                    <label class="block text-gray-600 text-sm font-semibold mb-1">Категория</label>
                    <input type="text" name="category" placeholder="Например: Дома 🏠" required 
                           class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-pink-500">
                </div>
                <div>
                    <label class="block text-gray-600 text-sm font-semibold mb-1">Что делать</label>
                    <textarea name="description" placeholder="Опишите идею подробно..." required rows="3"
                              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-pink-500"></textarea>
                </div>
                <button type="submit" 
                        class="w-full bg-pink-500 text-white font-bold py-2 px-4 rounded-lg hover:bg-pink-600 transition duration-200">
                    Сохранить идею
                </button>
            </form>
        </div>

        <!-- Список идей -->
        <div class="bg-white p-6 rounded-lg shadow-md">
            <h2 class="text-xl font-semibold text-gray-700 mb-4">Существующие идеи (Всего: {{len .}})</h2>
            <div class="space-y-4">
                {{range .}}
                <div class="border-b pb-4 last:border-b-0 last:pb-0 flex justify-between items-start">
                    <div class="pr-4">
                        <span class="inline-block bg-pink-100 text-pink-800 text-xs px-2 py-1 rounded font-semibold mb-1">
                            {{.Category}}
                        </span>
                        <p class="text-gray-700 text-sm">{{.Description}}</p>
                    </div>
                    <form action="/admin/delete" method="POST" class="flex-shrink-0">
                        <input type="hidden" name="id" value="{{.ID}}">
                        <button type="submit" class="text-red-500 hover:text-red-700 text-sm font-semibold">
                            Удалить
                        </button>
                    </form>
                </div>
                {{else}}
                <p class="text-gray-500 text-center">Идей пока нет. Добавьте первую!</p>
                {{end}}
            </div>
        </div>
    </div>
</body>
</html>
`

type DateIdea struct {
	gorm.Model
	Category    string
	Description string
}
type Event struct {
	gorm.Model
	UserID    int64 `gorm:"index"`
	Title     string
	EventDate string
}

var DB *gorm.DB

const (
	StateNone         = 0
	StateWaitingTitle = 1
	StateWaitingDate  = 2
)

var userStates = make(map[int64]int)

var userTempTitles = make(map[int64]string)

func initDB() {
	var err error

	DB, err = gorm.Open(sqlite.Open("bot.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}

	err = DB.AutoMigrate(&DateIdea{}, &Event{})
	if err != nil {
		log.Fatalf("Ошибка миграции БД: %v", err)
	}

	log.Println("База данных успешно инициализирована!")

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
	http.HandleFunc("/admin", adminHandler)
	http.HandleFunc("/admin/add", addIdeaHandler)
	http.HandleFunc("/admin/delete", deleteIdeaHandler)

	go func() {
		log.Println("Веб-сервер админки запущен на http://localhost:8080/admin")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Не удалось запустить веб-сервер: %v", err)
		}
	}()

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

			case "menu_remind":
				userStates[update.CallbackQuery.Message.Chat.ID] = StateWaitingTitle
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Введите название важного события 📝\n(например: Годовщина или День рождения любимой)")
				bot.Send(msg)

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
				bot.Send(editMsg)
			}

			continue
		}

		if update.Message != nil {
			chatID := update.Message.Chat.ID
			state := userStates[chatID]

			userName := update.Message.From.UserName
			if userName == "" {
				userName = update.Message.From.FirstName
			}
			log.Printf("[%s] написал: %s", userName, update.Message.Text)

			if update.Message.IsCommand() {
				var replyText string

				switch update.Message.Command() {
				case "start":
					userStates[chatID] = StateNone

					replyText = "Привет! Добро пожаловать в Date Romantic Bot 👩‍❤️‍👨\n\nЯ помогу тебе не забыть про важные даты и подкину крутые идеи для свиданий!"
					msg := tgbotapi.NewMessage(chatID, replyText)

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

				msg := tgbotapi.NewMessage(chatID, replyText)
				bot.Send(msg)
				continue
			}

			switch state {
			case StateWaitingTitle:
				userTempTitles[chatID] = update.Message.Text
				userStates[chatID] = StateWaitingDate

				msg := tgbotapi.NewMessage(chatID, "Отлично! Теперь введите дату в формате ДД.ММ 📅\n(например: 18.07)")
				bot.Send(msg)

			case StateWaitingDate:
				title := userTempTitles[chatID]
				dateStr := update.Message.Text

				re := regexp.MustCompile(`^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])$`)
				if !re.MatchString(dateStr) {
					msg := tgbotapi.NewMessage(chatID, "❌ Неверный формат даты! \nПожалуйста, введите дату строго в формате ДД.ММ (например: 18.07 или 02.12).")
					bot.Send(msg)
					continue
				}

				currentYear := time.Now().Year()

				fullDateStr := fmt.Sprintf("%s.%d", dateStr, currentYear)

				_, err := time.Parse("02.01.2006", fullDateStr)
				if err != nil {
					msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Такой даты не существует в %d году! Проверьте количество дней в месяце или февраль. 😉", currentYear))
					bot.Send(msg)
					continue
				}

				newEvent := Event{
					UserID:    chatID,
					Title:     title,
					EventDate: dateStr,
				}
				DB.Create(&newEvent)

				userStates[chatID] = StateNone
				delete(userTempTitles, chatID)

				msg := tgbotapi.NewMessage(chatID, "🎉 Успешно! Я запомнил эту дату и обязательно напомню о ней.")
				bot.Send(msg)
			default:
				msg := tgbotapi.NewMessage(chatID, "Ты написал обычный текст: "+update.Message.Text+"\nПожалуйста, пользуйся кнопками меню! 😉")
				bot.Send(msg)
			}
		}
	}
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	var list []DateIdea

	DB.Order("id desc").Find(&list)

	tmpl, err := template.New("admin").Parse(adminHTML)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, list)
}

func addIdeaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	category := r.FormValue("category")
	description := r.FormValue("description")

	if category != "" && description != "" {
		newIdea := DateIdea{
			Category:    category,
			Description: description,
		}
		DB.Create(&newIdea)
		log.Printf("[Админка] Добавлена новая идея: %s", category)
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func deleteIdeaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err == nil {
		// Удаляем из SQLite по ID
		DB.Delete(&DateIdea{}, id)
		log.Printf("[Админка] Удалена идея с ID: %d", id)
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
