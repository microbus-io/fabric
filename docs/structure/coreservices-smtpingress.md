# Package `coreservices/smtpingress`

The SMTP ingress microservice listens on port 25 for incoming email messages. An app can listen to the appropriate event in order to process and act upon the email message.

Use the following event sink in `service.yaml` to listen to the event:

```yaml
sinks:
  - signature: OnIncomingEmail(mailMessage *Email)
    description: OnIncomingEmail is triggered when a new email message is received.
    source: github.com/microbus-io/fabric/coreservices/smtpingress
```
