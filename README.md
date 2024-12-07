# 📱 YourFeed - Personal Telegram Feed

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge)](LICENSE)

</div>

YourFeed is a Telegram user-bot that aggregates your favorite channels into one personalized news feed and sends it to your main account. It acts like an aggregator of content, just like your Instagram or Facebook feed, but for Telegram.

## ✨ Features

- 🔄 **Unified Feed**: All posts from subscribed channels in one place
- 🚀 **Instant Updates**: Receive new posts immediately after publication
- 🎯 **Smart Filtering**: Skip comments and promotional posts
- 📊 **Efficient Processing**: Queue-based system for reliable message delivery
- 🛡️ **Error Protection**: Automatic retries on failures
- 💪 **High Performance**: Asynchronous message processing

## 🛠️ Technologies

- [Go](https://golang.org/) - Modern and fast programming language
- [gotd/td](https://github.com/gotd/td) - Telegram MTProto client in Go
- [uber-go/zap](https://github.com/uber-go/zap) - Fast, structured logger

## 📋 Requirements

- Go 1.22 or higher
- Telegram API credentials (API_ID and API_HASH)
- Access to Telegram channels
- Telegram account for user-bot

## 🚀 Quick Start

1. **Clone the repository**

   ```bash
   git clone https://github.com/yourusername/yourfeed.git
   cd yourfeed
   ```

2. **Configure the application**

   Create a `.env` file in the root directory:

   ```env
   API_ID=your_api_id
   API_HASH=your_api_hash
   PHONE=your_phone_number
   RECIPIENT_USERNAME=target_username
   ```

3. **Install dependencies**

   ```bash
   go mod download
   ```

4. **Build and run**
   ```bash
   go build -o bot ./cmd/bot
   ./bot
   ```

## 📁 Project Structure

```
.
├── cmd/
│   └── bot/
│       └── main.go           # Application entry point
├── internal/
│   ├── handler/
│   │   └── handler.go        # Telegram updates handler
│   └── queue/
│       └── queue.go          # Message queue
├── .env                      # Configuration
├── go.mod                    # Go dependencies
└── README.md                 # Documentation
```

## 🔧 Configuration

| Variable           | Description                    |
| ------------------ | ------------------------------ |
| API_ID             | Telegram application ID        |
| API_HASH           | Telegram application hash      |
| PHONE              | Phone number for authorization |
| RECIPIENT_USERNAME | Target username for messages   |

## 🤝 Contributing

We welcome contributions to the project! Here's how you can help:

1. Fork the repository
2. Create a branch for your changes
3. Make changes and commit them
4. Submit a pull request

## 📝 TODO

- [ ] Add support for multiple recipients
- [ ] Implement keyword-based filtering
- [ ] Add web interface for management

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 👥 Authors

- [@pav-kh](https://github.com/pav-kh) - Idea and implementation

---

<div align="center">
⭐ Star this project if you find it useful! ⭐
</div>
