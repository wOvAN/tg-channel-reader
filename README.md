# tg-channel-reader

Чтение сообщений из открытых Telegram-каналов через публичный HTML-архив (`t.me/s/<username>`).

**Без авторизации, без API-ключей, без ботов.**

## Установка

```bash
go get github.com/wOvAN/tg-channel-reader
```

## Использование

```go
import tg "github.com/wOvAN/tg-channel-reader"

reader := tg.New("durov")
msgs, err := reader.Fetch(context.Background(), 10)
```

### Опции

```go
reader := tg.New("durov",
    tg.WithSince(time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)),
    tg.WithUntil(time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)),
    tg.WithProxy("http://proxy:8080"),
    tg.WithTimeout(2 * time.Minute),
)
```

| Опция | Описание |
|-------|----------|
| `WithHTTPClient(client)` | Кастомный HTTP-клиент |
| `WithProxy(url)` | HTTP(S)/SOCKS5 прокси |
| `WithSince(date)` | Только от этой даты (включительно) |
| `WithUntil(date)` | Только до этой даты (включительно) |
| `WithTimeout(d)` | Таймаут запроса (по умолчанию 30s) |

### API

```go
// Получить сообщения (newest first)
msgs, err := reader.Fetch(context.Background(), 10)

// Получить сообщения и инфо о канале
msgs, info, err := reader.FetchWithInfo(context.Background(), 10)

// Только инфо о канале
info, err := reader.FetchInfo(context.Background())
```

### Типы

```go
type Message struct {
    ID             int          // номер сообщения
    Channel        string       // username канала
    Date           time.Time    // время публикации (UTC)
    Text           string       // текст сообщения
    MediaType      string       // тип: text, photo, video, sticker, и др.
    Author         string       // автор (для каналов с несколькими авторами)
    Views          string       // просмотры ("7.62K", "57K")
    ViewsCount     int          // просмотры как число (7620, 57000)
    Edited         bool         // отредактировано
    Reactions      []Reaction   // реакции
    ForwardedFrom  string       // от кого переслано
    ForwardedFromURL string     // ссылка на источник
    URLs           []string     // inline-ссылки в тексте
    LinkPreview    *LinkPreview // превью ссылки
    Link           string       // ссылка на сообщение
}

type Reaction struct {
    Emoji string // "👍", "❤", ...
    Count int    // количество
}

type LinkPreview struct {
    URL         string // ссылка
    SiteName    string // название сайта
    Title       string // заголовок
    Description string // описание
    ImageURL    string // картинка превью
}

type ChannelInfo struct {
    Title       string // название канала
    Description string // описание
    ImageURL    string // аватар
    Subscribers string // подписчики ("10.4M")
    Photos      int    // количество фото
    Videos      int    // количество видео
    Links       int    // количество ссылок
}
```

## Примеры

См. [examples/](examples/) — каждый пример — самостоятельный проект:

- [basic](examples/basic/) — простое чтение и вывод
- [filter](examples/filter/) — фильтр по дате + агрегация реакций
- [json](examples/json/) — JSON-вывод для piping
- [proxy](examples/proxy/) — чтение через прокси
- [stats](examples/stats/) — статистика канала

## Тесты

```bash
go test -v
```

## Ограничения

- Работает только с **публичными** каналами (у которых есть `t.me/s/<username>`)
- Telegram ограничивает глубину архива — обычно доступны последние ~3000 сообщений
- При частых запросах Telegram может вернуть CAPTCHA (429/403)
