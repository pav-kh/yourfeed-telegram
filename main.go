package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

func main() {
	// Замените на ваш API ID и API Hash
	apiID := int(21405433)                        // Ваш API ID
	apiHash := "db137fe971b2bd07d0d26280fed0ab4d" // Ваш API Hash

	client := telegram.NewClient(apiID, apiHash, telegram.Options{})

	ctx := context.Background()

	// Запуск клиента
	if err := client.Run(ctx, func(ctx context.Context) error {
		// Функция для ввода кода аутентификации
		codePrompt := func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
			// Введите код
			fmt.Print("Введите код: ")
			code, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(code), nil
		}

		// Процесс аутентификации
		phone := "+995568652771" // Ваш номер телефона
		if err := auth.NewFlow(
			auth.CodeOnly(phone, auth.CodeAuthenticatorFunc(codePrompt)),
			auth.SendCodeOptions{},
		).Run(ctx, client.Auth()); err != nil {
			return err
		}

		// Получение канала
		channelUsername := "cbpub" // Замените на имя пользователя канала
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
			return fmt.Errorf("Канал не найден")
		}

		// Получение последнего сообщения
		messages, err := client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:  channel,
			Limit: 1,
		})
		if err != nil {
			return err
		}

		history, ok := messages.(*tg.MessagesChannelMessages)
		if !ok || len(history.Messages) == 0 {
			return fmt.Errorf("Сообщения не найдены")
		}

		message, ok := history.Messages[0].(*tg.Message)
		if !ok {
			return fmt.Errorf("Неверный тип сообщения")
		}

		// Получение пользователя
		userRes, err := client.API().ContactsResolveUsername(ctx, "pashamakeme") // Замените на имя пользователя получателя
		if err != nil {
			return err
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
			return fmt.Errorf("Пользователь не найден")
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
			return err
		}

		fmt.Println("Сообщение успешно переслано")
		return nil
	}); err != nil {
		log.Fatal(err)
	}
}
