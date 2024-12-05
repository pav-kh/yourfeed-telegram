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

// Обработчик обновлений
type updateHandler struct {
	client        *telegram.Client
	channels      map[int64]*tg.InputPeerChannel
	inputPeerUser tg.InputPeerClass
}

func (h *updateHandler) Handle(ctx context.Context, u tg.UpdatesClass) error {
	switch updates := u.(type) {
	case *tg.Updates:
		for _, update := range updates.Updates {
			h.processUpdate(ctx, update)
		}
	case *tg.UpdatesCombined:
		for _, update := range updates.Updates {
			h.processUpdate(ctx, update)
		}
	case *tg.UpdateShort:
		h.processUpdate(ctx, updates.Update)
	default:
		log.Printf("Необработанный тип обновления: %T", updates)
	}
	return nil
}

func (h *updateHandler) processUpdate(ctx context.Context, update tg.UpdateClass) {
	switch u := update.(type) {
	case *tg.UpdateNewChannelMessage:
		if err := h.handleUpdate(ctx, u); err != nil {
			log.Printf("Ошибка при обработке обновления: %v", err)
		}
	default:
		// Игнорируем другие типы обновлений
	}
}

func (h *updateHandler) handleUpdate(ctx context.Context, update *tg.UpdateNewChannelMessage) error {
	message, ok := update.Message.(*tg.Message)
	if !ok {
		return nil
	}

	peerChannel, ok := message.PeerID.(*tg.PeerChannel)
	if !ok {
		return nil
	}

	// Проверяем, является ли канал тем, который мы отслеживаем
	channel, exists := h.channels[peerChannel.ChannelID]
	if !exists {
		return nil
	}

	// Игнорируем комментарии к постам
	if message.ReplyTo != nil && message.Post {
		log.Println("Комментарий к посту не будет переслан")
		return nil
	}

	// Пересылаем сообщение
	rand.Seed(time.Now().UnixNano())
	randomID := rand.Int63()
	_, err := h.client.API().MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
		FromPeer: &tg.InputPeerChannel{
			ChannelID:  peerChannel.ChannelID,
			AccessHash: channel.AccessHash,
		},
		ID:                []int{message.ID},
		RandomID:          []int64{randomID},
		ToPeer:            h.inputPeerUser,
		Silent:            true,
		Background:        false,
		WithMyScore:       false,
		DropAuthor:        false,
		DropMediaCaptions: false,
	})
	if err != nil {
		log.Printf("Ошибка при пересылке сообщения из канала %d: %v", peerChannel.ChannelID, err)
		return err
	}

	// Помечаем сообщение как прочитанное
	_, err = h.client.API().ChannelsReadHistory(ctx, &tg.ChannelsReadHistoryRequest{
		Channel: &tg.InputChannel{
			ChannelID:  peerChannel.ChannelID,
			AccessHash: channel.AccessHash,
		},
		MaxID: message.ID,
	})
	if err != nil {
		log.Printf("Ошибка при отметке сообщения как прочитанного для канала %d: %v", peerChannel.ChannelID, err)
		return err
	}

	log.Printf("Сообщение из канала %d успешно переслано и помечено как прочитанное", peerChannel.ChannelID)
	return nil
}

func main() {
	// Загрузка переменных окружения из .env файла
	if err := godotenv.Load(); err != nil {
		log.Fatal("Ошибка загрузки файла .env")
	}

	// Получение значений из .env файла
	apiID, err := strconv.Atoi(os.Getenv("API_ID"))
	if err != nil {
		log.Fatal(err)
	}
	apiHash := os.Getenv("API_HASH")
	phone := os.Getenv("PHONE")
	recipientUsername := os.Getenv("RECIPIENT_USERNAME")

	ctx := context.Background()

	// Создаем клиент и передаем обработчик обновлений через опции
	handler := &updateHandler{}

	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		UpdateHandler:  handler,
		SessionStorage: &telegram.FileSessionStorage{Path: "session.json"},
	})

	// Запуск клиента
	if err := client.Run(ctx, func(ctx context.Context) error {
		// Функция для ввода кода аутентификации
		codePrompt := func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
			var code string
			fmt.Print("Введите код: ")
			_, err := fmt.Scanln(&code)
			return code, err
		}

		// Процесс аутентификации
		flow := auth.NewFlow(
			auth.CodeOnly(phone, auth.CodeAuthenticatorFunc(codePrompt)),
			auth.SendCodeOptions{},
		)
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			if rpcErr, ok := err.(*telegram.Error); ok && rpcErr.Code == 420 {
				waitTime := rpcErr.Argument
				log.Printf("FLOOD_WAIT: необходимо подождать %d секунд перед повторной попыткой", waitTime)
				time.Sleep(time.Duration(waitTime) * time.Second)
				return client.Auth().IfNecessary(ctx, flow)
			}
			return fmt.Errorf("ошибка аутентификации: %w", err)
		}

		// Получение списка каналов
		dialogs, err := client.API().MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			Limit:      100,
			OffsetDate: 0,
			OffsetID:   0,
			OffsetPeer: &tg.InputPeerEmpty{},
		})
		if err != nil {
			return fmt.Errorf("ошибка при получении диалогов: %v", err)
		}

		var chats []tg.ChatClass
		switch d := dialogs.(type) {
		case *tg.MessagesDialogs:
			chats = d.Chats
		case *tg.MessagesDialogsSlice:
			chats = d.Chats
		default:
			return fmt.Errorf("неверный тип данных для диалогов: %T", dialogs)
		}

		channels := make(map[int64]*tg.InputPeerChannel)
		for _, chat := range chats {
			if ch, ok := chat.(*tg.Channel); ok && !ch.Megagroup {
				channels[ch.ID] = &tg.InputPeerChannel{
					ChannelID:  ch.ID,
					AccessHash: ch.AccessHash,
				}
			}
		}

		if len(channels) == 0 {
			return fmt.Errorf("нет подписанных каналов")
		}

		// Получение пользователя для пересылки сообщений
		userRes, err := client.API().ContactsResolveUsername(ctx, recipientUsername)
		if err != nil {
			return fmt.Errorf("ошибка при получении пользователя: %v", err)
		}

		var inputPeerUser tg.InputPeerClass
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
			return fmt.Errorf("пользователь не найден")
		}

		// Инициализируем обработчик обновлений
		handler.client = client
		handler.channels = channels
		handler.inputPeerUser = inputPeerUser

		log.Println("Бот запущен и ожидает новых сообщений...")

		// Ожидание завершения контекста
		<-ctx.Done()
		return nil
	}); err != nil {
		log.Fatal(err)
	}
}
