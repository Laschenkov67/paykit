Исправил найденные баги — от критичных финансовых/security-проблем до архитектурных и code-quality дефектов:

Критичные (деньги/безопасность):

YooKassa: Capture: req.Capture || true — двухстадийные платежи были невозможны в принципе (всегда списывало сразу). Переименовал поле CreatePaymentRequest.Capture → TwoStage (нулевое значение false теперь соответствует документированному поведению по умолчанию) и прокинул его во все провайдеры (Tinkoff PayType, Stripe capture_method=manual, CloudPayments RequireConfirmation).
YooKassa webhook: ParseWebhook вообще не проверял подлинность запроса — AllowedIPs() существовала, но не вызывалась. Любой мог подделать payment.succeeded. Добавил реальную проверку r.RemoteAddr по CIDR-диапазонам.
Tinkoff Refund: Token подписывался до добавления Amount в параметры — частичные возвраты гарантированно отклонялись бы API из-за неверной подписи. Исправил порядок.
Tinkoff/Robokassa: сравнение подписи вебхука было не constant-time (==/!=) — timing attack. Заменил на internal/signing.EqualHex.
CloudPayments: int64(floatAmount * 100) усекало копейки из-за погрешности float64 (19.99 → 1998 вместо 1999) — и в GetPayment, и в webhook. Переключил на парсинг через строку с фиксированной точностью (paykit.ParseMajor).
Manager.HandleWebhook: всегда отвечал пустым 200 OK, хотя Tinkoff требует тело "OK", Robokassa — "OK"+InvId, CloudPayments — {"code":0}. Без этого провайдеры считают вебхук неуспешным и шлют его повторно бесконечно. Добавил WebhookAck в интерфейс Provider и реализовал у каждого.
money.go ParseMajor: суммы вида -0.50 теряли знак (получали +50 вместо -50).
Архитектура / качество кода

Удалил самописные base64-кодировщики в YooKassa и CloudPayments → encoding/base64.
Удалил ручные циклы чтения тела ответа в Robokassa/Stripe → io.ReadAll.
Заменил хрупкий поиск подстроки в XML-ответе Robokassa на нормальный encoding/xml.Unmarshal.
Подключил ранее полностью неиспользуемый internal/signing ко всем провайдерам, где он применим.
Убрал мёртвый код (var _ = json.Marshal, неиспользуемые импорты, мёртвую переменную в тестовом хелпере).
Добавил/обновил регрессионные тесты под каждый найденный баг (signature-over-amount у Tinkoff, IP-фильтрация YooKassa, точность сумм CloudPayments, ack-тело Manager, TwoStage-семантика).