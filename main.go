package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/joho/godotenv"
)

func main() {
	// Загрузка переменных окружения из .env файла
	if err := godotenv.Load(); err != nil {
		log.Fatal("Ошибка загрузки файла .env")
	}

	// Получение значений из .env файла
	apiID, err := strconv.Atoi(os.Getenv("API_ID"))
	if err != nil {
		log.Fatal("Неверный apiID")
	}
	apiHash := os.Getenv("API_HASH")
	phone := os.Getenv("PHONE")
	channelUsername := os.Getenv("CHANNEL_USERNAME")
	userUsername := os.Getenv("USER_USERNAME")

	client := telegram.NewClient(apiID, apiHash, telegram.Options{})

	ctx := context.Background()

	// Запуск клиента
	if err := client.Run(ctx, func(ctx context.Context) error {
		// Функция для ввода кода аутентификации
		codePrompt := func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
			// Введите код
			var code string
			fmt.Print("Введите код: ")
			_, err := fmt.Scanln(&code)
			if err != nil {
				return "", err
			}
			return code, nil
		}

		// Процесс аутентификации
		if err := auth.NewFlow(
			auth.CodeOnly(phone, auth.CodeAuthenticatorFunc(codePrompt)),
			auth.SendCodeOptions{},
		).Run(ctx, client.Auth()); err != nil {
			return err
		}

		// Получение канала
		res, err := client.API().ContactsResolveUsername(ctx, channelUsername)
		if err != nil {
			return err
		}

		var channel *tg.InputPeerChannel
		for _, chat := range res.Chats {
			if ch, ok := chat.(*tg.Channel); ok {
				channel = &tg.InputPeerChannel{
					ChannelID:  ch.ID,
					AccessHash: ch.AccessHash,
				}
				break
			}
		}

		if channel == nil {
			return fmt.Errorf("канал не найден")
		}

		var lastMessageID int

		// Настройка мониторинга канала на наличие новых сообщений
		go func() {
			for {
				messages, err := client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
					Peer:  channel,
					Limit: 1,
				})
				if err != nil {
					log.Printf("Ошибка при получении истории сообщений: %v", err)
					continue
				}

				history, ok := messages.(*tg.MessagesChannelMessages)
				if !ok || len(history.Messages) == 0 {
					log.Println("Сообщения не найдены")
					continue
				}

				message, ok := history.Messages[0].(*tg.Message)
				if !ok {
					log.Println("Неверный тип сообщения")
					continue
				}

				if message.ID == lastMessageID {
					log.Println("Новое сообщение отсутствует")
					time.Sleep(30 * time.Second)
					continue
				}

				lastMessageID = message.ID

				// Получение пользователя для пересылки сообщения
				userRes, err := client.API().ContactsResolveUsername(ctx, userUsername) // Замените на имя пользователя получателя
				if err != nil {
					log.Printf("Ошибка при получении пользователя: %v", err)
					continue
				}

				var inputPeerUser *tg.InputPeerUser
				for _, user := range userRes.Users {
					if usr, ok := user.(*tg.User); ok {
						inputPeerUser = &tg.InputPeerUser{
							UserID:     usr.ID,
							AccessHash: usr.AccessHash,
						}
						break
					}
				}

				if inputPeerUser == nil {
					log.Println("Пользователь не найден")
					continue
				}

				// Подготовка данных для пересылки сообщения
				rand.Seed(time.Now().UnixNano())
				randomID := rand.Int63()

				// Пересылка сообщения пользователю
				_, err = client.API().MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
					FromPeer: channel,
					ID:       []int{message.ID},
					RandomID: []int64{randomID},
					ToPeer:   inputPeerUser,
				})
				if err != nil {
					log.Printf("Ошибка при пересылке сообщения: %v", err)
					continue
				}

				log.Println("Сообщение успешно переслано")

				time.Sleep(30 * time.Second) // Задержка перед следующей проверкой
			}
		}()

		<-ctx.Done()
		return nil
	}); err != nil {
		log.Fatal(err)
	}
}
