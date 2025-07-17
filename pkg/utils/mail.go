package utils

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/gomail.v2"

	"github.com/mailjet/mailjet-apiv3-go/v4"
)

// SendMailMailjet отправляет письмо через Mailjet
func SendMailMailjet(tgID int64, qrcode string, summa float64, crypto float64) {
	fmt.Println("Отправка письма через Mailjet")
	fromEmail := "themountok@icloud.com" // ваш подтверждённый отправитель в Mailjet
	toEmail := "themountok@gmail.com"    // получатель
	subject := "У вас новый заказ!"

	body := fmt.Sprintf(`<body style="margin:0;padding:0;background:#f6f6f6;">
  <table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f6f6f6;min-height:100vh;">
    <tr>
      <td align="center">
        <table width="100%%" cellpadding="0" cellspacing="0" border="0" style="max-width:600px;background:#f6f6f6;">
          <tr>
            <td style="padding:32px 0;">
              <table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f3f2f0;border-radius:28px;padding:0 0 32px 0;">
                <tr>
                  <td style="padding:32px 32px 0 32px;text-align:left;">
                    <h1 style="margin:0 0 12px 0;font-family:Arial,sans-serif;font-size:32px;font-weight:700;line-height:1.2;color:#111;">Оплата заказа</h1>
                    <p style="margin:0 0 24px 0;font-family:Arial,sans-serif;font-size:20px;line-height:1.4;color:#222;">Пожалуйста, оплатите заказ по данным ниже.</p>
                    <table cellpadding="0" cellspacing="0" border="0" style="width:100%%;margin-bottom:24px;">
                      <tr>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#555;padding:6px 0;">Telegram ID:</td>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#111;font-weight:bold;padding:6px 0;">%d</td>
                      </tr>
                      <tr>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#555;padding:6px 0;">Сумма к оплате (₽):</td>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#111;font-weight:bold;padding:6px 0;">%f</td>
                      </tr>
                      <tr>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#555;padding:6px 0;">Сумма в USDT:</td>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#111;font-weight:bold;padding:6px 0;">%f</td>
                      </tr>
                    </table>
                    
                    <div style="text-align:center;">
                      <a href="https://platapay.ru/admin/manual-pay" style="display:inline-block;padding:18px 0;width:100%%;max-width:320px;background:#111;color:#fff;font-family:Arial,sans-serif;font-size:22px;font-weight:600;text-decoration:none;border-radius:20px;">Оплатить</a>
                    </div>
                  </td>
                </tr>
              </table>
              <div style="text-align:center;font-family:Arial,sans-serif;font-size:13px;color:#aaa;margin-top:24px;">Если у вас возникли вопросы, свяжитесь с поддержкой.</div>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>`, tgID, summa, crypto)

	apiKey := os.Getenv("MAILJET_API_KEY")
	secretKey := os.Getenv("MAILJET_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		log.Println("MAILJET_API_KEY или MAILJET_SECRET_KEY не установлены!")
		return
	}

	mj := mailjet.NewMailjetClient(apiKey, secretKey)
	messagesInfo := []mailjet.InfoMessagesV31{
		{
			From: &mailjet.RecipientV31{
				Email: fromEmail,
				Name:  "EVOCAR",
			},
			To: &mailjet.RecipientsV31{
				{
					Email: toEmail,
					Name:  "Получатель",
				},
			},
			Subject:  subject,
			HTMLPart: body,
		},
	}
	messages := &mailjet.MessagesV31{Info: messagesInfo}
	res, err := mj.SendMailV31(messages)
	if err != nil {
		log.Println("Ошибка при отправке письма через Mailjet:", err)
	} else {
		log.Printf("Mailjet ответ: %+v", res)
		log.Printf("Письмо через Mailjet отправлено: ордер для %d на сумму %f в usdt - %f", tgID, summa, crypto)
	}
}

func SendMail(tgID int64, qrcode string, summa float64, crypto float64) {
	fmt.Println("Отправка письма")
	// Настройки
	from := "themountok@gmail.com"              // ← твоя почта Gmail
	password := os.Getenv("GMAIL_APP_PASSWORD") // ← пароль приложения (16 символов без пробелов)
	to := "dashuev@internet.ru"                 // ← кому отправляем
	subject := "У вас новый заказ!"             // ← тема письма

	body := fmt.Sprintf(`
    <body style="margin:0;padding:0;background:#f6f6f6;">
  <table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f6f6f6;min-height:100vh;">
    <tr>
      <td align="center">
        <table width="100%%" cellpadding="0" cellspacing="0" border="0" style="max-width:600px;background:#f6f6f6;">
          <tr>
            <td style="padding:32px 0;">
              <table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f3f2f0;border-radius:28px;padding:0 0 32px 0;">
                <tr>
                  <td style="padding:32px 32px 0 32px;text-align:left;">
                    <h1 style="margin:0 0 12px 0;font-family:Arial,sans-serif;font-size:32px;font-weight:700;line-height:1.2;color:#111;">Оплата заказа</h1>
                    <p style="margin:0 0 24px 0;font-family:Arial,sans-serif;font-size:20px;line-height:1.4;color:#222;">Пожалуйста, оплатите заказ по данным ниже.</p>
                    <table cellpadding="0" cellspacing="0" border="0" style="width:100%%;margin-bottom:24px;">
                      <tr>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#555;padding:6px 0;">Telegram ID:</td>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#111;font-weight:bold;padding:6px 0;">%d</td>
                      </tr>
                      <tr>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#555;padding:6px 0;">Сумма к оплате (₽):</td>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#111;font-weight:bold;padding:6px 0;">%f</td>
                      </tr>
                      <tr>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#555;padding:6px 0;">Сумма в USDT:</td>
                        <td style="font-family:Arial,sans-serif;font-size:16px;color:#111;font-weight:bold;padding:6px 0;">%f</td>
                      </tr>
                    </table>
                    
                    <div style="text-align:center;">
                      <a href="https://platapay.ru/admin/manual-pay" style="display:inline-block;padding:18px 0;width:100%%;max-width:320px;background:#111;color:#fff;font-family:Arial,sans-serif;font-size:22px;font-weight:600;text-decoration:none;border-radius:20px;">Оплатить</a>
                    </div>
                  </td>
                </tr>
              </table>
              <div style="text-align:center;font-family:Arial,sans-serif;font-size:13px;color:#aaa;margin-top:24px;">Если у вас возникли вопросы, свяжитесь с поддержкой.</div>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
	`, tgID, summa, crypto)

	// Создаём сообщение
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	m.SetHeader("Reply-To", "themountok@gmail.com")

	// Настраиваем SMTP-доставку
	d := gomail.NewDialer("smtp.gmail.com", 587, from, password)

	// Отправка
	if err := d.DialAndSend(m); err != nil {
		log.Println("Ошибка при отправке письма:", err)
	} else {
		log.Printf("Письмо через Mailjet отправлено: ордер для %d на сумму %f в usdt - %f", tgID, summa, crypto)
	}
}
