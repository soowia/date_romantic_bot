package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// 1. Берем токен из переменных окружения (для безопасности)
	botToken := os.Getenv("TELEGRAM_APITOKEN")
	if botToken == "" {
		log.Fatal("Переменная окружения TELEGRAM_APITOKEN не установлена!")
	}

	// 2. Инициализируем бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err) // Если токен неверный или сети нет, программа упадет с ошибкой
	}

	// Включаем дебаг-режим, чтобы в консоли было видно все входящие сообщения
	bot.Debug = true

	log.Printf("Авторизовались под аккаунтом %s", bot.Self.UserName)

	// 3. Настраиваем Long Polling (получение обновлений от Telegram)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Получаем Go-канал, в который будут прилетать новые сообщения
	updates := bot.GetUpdatesChan(u)

	// 4. Бесконечный цикл обработки сообщений
	for update := range updates {
		// Если пришло не текстовое сообщение (а, например, пользователь удалил чат) — игнорируем
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] написал: %s", update.Message.From.UserName, update.Message.Text)

		// Создаем ответное сообщение: повторяем то, что написал пользователь
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ты написал: "+update.Message.Text)

		// Отправляем ответ обратно в Telegram
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Не удалось отправить сообщение: %v", err)
		}
	}
}
