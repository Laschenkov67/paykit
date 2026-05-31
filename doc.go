// Пакет paykit предоставляет унифицированный, идиоматический интерфейс Go для самых популярных
// платежных систем, используемых в России, СНГ и по всему миру: YooKassa, Tinkoff,
// CloudPayments, Robokassa и Stripe.

//
// Библиотека предоставляет единый интерфейс Provider, который абстрагирует создание платежей,
// захват, отмену, возврат средств и анализ веб-хуков. Каждая интеграция с платежной системой находится в своем
// собственном подпакете в папке providers/ и может быть импортирована независимо, поэтому
// итоговый исполняемый файл содержит только тот код, который вы фактически используете.

// //
// Быстрый старт:
//
// import (
// "github.com/laschenkov67/paykit"
// "github.com/laschenkov67/paykit/providers/yookassa"
// )
//
// p, _ := yookassa.New("shopID", "secret")
// pay, err := p.CreatePayment(ctx, paykit.CreatePaymentRequest{
// OrderID: "order-42",
// Amount: paykit.RUB(199_00), // 199 ₽
// Description: "T-Shirt",
// ReturnURL: "https://shop.example.com/return",
// })
//
// См. каталог examples/ для сквозной интеграции с веб-хуками.
package paykit
