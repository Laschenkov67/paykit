# paykit

[![Go Reference](https://pkg.go.dev/badge/github.com/laschenkov67/paykit.svg)](https://pkg.go.dev/github.com/yourname/paykit)
[![CI](https://github.com/yourname/paykit/actions/workflows/ci.yml/badge.svg)](https://github.com/yourname/paykit/actions)

Единый, идиоматичный Go-интерфейс для популярных платёжных провайдеров:
**YooKassa**, **Tinkoff Acquiring**, **CloudPayments**, **Robokassa**, **Stripe**.

## Установка

    go get github.com/laschenkov67/paykit

## Быстрый старт

См. examples/.

## Сравнение возможностей

| Возможность      | YooKassa | Tinkoff | CloudPayments | Robokassa | Stripe |
|------------------|:--------:|:-------:|:-------------:|:---------:|:------:|
| CreatePayment    | ✅       | ✅      | ✅            | ✅        | ✅     |
| GetPayment       | ✅       | ✅      | ✅            | ✅        | ✅     |
| Two-stage hold   | ✅       | ✅      | ✅            | —         | ✅     |
| Refund (API)     | ✅       | ✅      | ✅            | —         | ✅     |
| Webhook signed   | IP-based | HMAC    | HMAC          | MD5       | HMAC   |

## Лицензия — MIT.