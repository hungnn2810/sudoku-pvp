# RABBITMQ_TOPOLOGY.md

Exchange:

game.events

Type:

topic

---

Queues

ranking.queue
wallet.queue
mission.queue
analytics.queue
notification.queue

---

Routing Keys

match.started
match.finished

wallet.transaction.created

ranking.changed

mission.completed

shop.purchased

---

Consumers

Ranking Service
Wallet Service
Mission Service
Analytics Service
Notification Service

---

Retry Strategy

3 retries

Dead Letter Exchange:

game.dlx

Dead Letter Queue:

game.dead.queue
