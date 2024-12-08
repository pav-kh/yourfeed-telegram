# 📱 YourFeed - Personal Telegram Feed

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge)](LICENSE)
![Status](https://img.shields.io/badge/status-active-success.svg?style=for-the-badge)

</div>

YourFeed is a Telegram user-bot that aggregates your favorite channels into one personalized news feed and sends it to your main account. It acts like an aggregator of content, just like your Instagram or Facebook feed, but for Telegram.

## 📑 Table of Contents

- [Features](#-features)
- [Technologies](#️-technologies)
- [Prerequisites](#-prerequisites)
- [Quick Start](#-quick-start)
- [Project Structure](#-project-structure)
- [Configuration](#-configuration)
- [Security](#-security)
- [Troubleshooting](#-troubleshooting)
- [Contributing](#-contributing)
- [TODO](#-todo)
- [License](#-license)
- [Authors](#-authors)
- [Contact](#-contact)

## ✨ Features

- 🔄 **Unified Feed**: All posts from subscribed channels in one place
- 🚀 **Instant Updates**: Receive new posts immediately after publication
- 🎯 **Smart Filtering**: Skip comments and promotional posts
- 📊 **Performance Metrics**: Real-time monitoring of message processing
- 🛡️ **Error Protection**: Automatic retries and rate limiting
- 💪 **High Performance**: Object pooling and worker-based processing
- 🔒 **Graceful Shutdown**: Clean shutdown with proper resource cleanup

## 🛠️ Technologies

- [Go](https://golang.org/) - Modern and fast programming language
- [gotd/td](https://github.com/gotd/td) - Telegram MTProto client in Go
- [godotenv](https://github.com/joho/godotenv) - Environment variable management

## 📋 Prerequisites

Before you begin, ensure you have:

- Go 1.22 or higher installed
- A Telegram account
- Telegram API credentials (API_ID and API_HASH) from https://my.telegram.org
- Access to Telegram channels you want to aggregate
- Git installed (for cloning the repository)

## 🚀 Quick Start

1. **Clone the repository**

   ```bash
   git clone https://github.com/pav-kh/yourfeed-telegram.git
   cd yourfeed-telegram
   ```

2. **Configure the application**

   Create a `.env` file based on `.env.example`:

   ```bash
   cp .env.example .env
   ```

   Edit `.env` with your credentials:

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
yourfeed-telegram/
├── cmd/
│   └── bot/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/              # Configuration management
│   │   └── config.go
│   ├── handler/             # Telegram updates handler
│   │   └── handler.go
│   ├── metrics/             # Performance metrics
│   │   └── metrics.go
│   ├── models/              # Data models
│   │   └── message.go
│   ├── pool/                # Object and worker pools
│   │   ├── object.go
│   │   └── worker.go
│   └── ratelimit/           # Rate limiting
│       └── ratelimit.go
├── pkg/
│   └── telegram/            # Telegram client wrapper
│       └── client.go
├── .env.example             # Example configuration
├── go.mod                   # Go dependencies
└── README.md               # Documentation
```

## 🔧 Configuration

| Variable           | Description                    | Example              |
| ------------------ | ------------------------------ | -------------------- |
| API_ID             | Telegram application ID        | 123456               |
| API_HASH           | Telegram application hash      | abcdef1234567890     |
| PHONE              | Phone number for authorization | +1234567890          |
| RECIPIENT_USERNAME | Target username for messages   | username (without @) |

## 🤝 Security

- Never share your `.env` file or API credentials
- Keep your session file (`session.json`) private
- Regularly update dependencies to patch security vulnerabilities
- The bot runs with user privileges, be cautious with channel access
- Consider using a separate Telegram account for the bot
- Monitor the bot's activity regularly for unusual behavior

## ❗ Troubleshooting

Common issues and solutions:

1. **Authentication Failed**

   - Verify API credentials in `.env`
   - Check phone number format
   - Ensure 2FA is handled correctly

2. **Rate Limiting**

   - The bot implements automatic rate limiting
   - Wait for the specified time before retrying
   - Consider reducing the number of channels

3. **Message Processing Issues**
   - Check logs for specific error messages
   - Verify channel permissions
   - Ensure recipient username is correct

## 🤝 Contributing

We welcome contributions! Here's how you can help:

1. **Fork & Clone**

   ```bash
   git clone https://github.com/yourusername/yourfeed-telegram.git
   ```

2. **Create Branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Changes**

   - Follow Go best practices
   - Add tests for new features
   - Update documentation

4. **Commit**

   ```bash
   git commit -m "feat: add your feature description"
   ```

5. **Push & Submit PR**
   ```bash
   git push origin feature/your-feature-name
   ```

Please read our [Contributing Guidelines](.github/CONTRIBUTING.md) for more details.

## 📝 TODO

- [ ] Add support for multiple recipients
- [ ] Implement keyword-based filtering
- [ ] Add web interface for management

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 👥 Authors

- [@pav-kh](https://github.com/pav-kh) - Idea and implementation

## 📞 Contact

- **Project Maintainer**: [@pav-kh](https://github.com/pav-kh)
- **Issue Tracker**: [GitHub Issues](https://github.com/pav-kh/yourfeed-telegram/issues)
- **Discussions**: [GitHub Discussions](https://github.com/pav-kh/yourfeed-telegram/discussions)

---

<div align="center">
⭐ Star this project if you find it useful! ⭐

[Report Bug](https://github.com/pav-kh/yourfeed-telegram/issues) · [Request Feature](https://github.com/pav-kh/yourfeed-telegram/issues)

</div>
