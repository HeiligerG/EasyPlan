# Konzept: Simple Cash Planner

## 1. Ziel

Eine extrem einfache Self-Hosted-Webanwendung zur Planung von wiederkehrenden Fixkosten und Einnahmen.

Die Anwendung ist **kein Haushaltsbuch** und **keine Buchhaltungssoftware**.

Sie soll insbesondere nicht erfassen:

* Einkäufe
* einzelne Kartentransaktionen
* Kategorien für Alltagsausgaben
* Banktransaktionen
* Belege
* Budgets pro Kategorie
* Vermögensentwicklung
* Bank-APIs

Stattdessen beantwortet sie hauptsächlich drei Fragen:

1. Welche festen Zahlungen kommen als Nächstes?
2. Wie viel Geld muss auf meinem Konto vorhanden sein?
3. Wie viel Geld ist aktuell wirklich frei verfügbar?

---

# 2. Grundprinzip

Der Benutzer pflegt nur:

* aktuellen Kontostand
* wiederkehrende Einnahmen
* wiederkehrende Ausgaben
* einmalige zukünftige Ausgaben

Beispiel:

| Typ      | Name             |     Betrag | Wiederholung | Datum       |
| -------- | ---------------- | ---------: | ------------ | ----------- |
| Einnahme | Lohn             | +7'000 CHF | monatlich    | 25.         |
| Ausgabe  | Miete            | -2'000 CHF | monatlich    | 1.          |
| Ausgabe  | Krankenkasse     |   -450 CHF | monatlich    | 5.          |
| Ausgabe  | Internet         |    -60 CHF | monatlich    | 15.         |
| Ausgabe  | Autoversicherung | -1'200 CHF | jährlich     | 15. Februar |
| Ausgabe  | Serafe           |   -335 CHF | jährlich     | 20. März    |

Der aktuelle Kontostand wird bei Bedarf manuell aktualisiert.

Es ist daher egal, wofür sonst Geld ausgegeben wurde.

Beispiel:

Gestern:

```
Kontostand: 8'420 CHF
```

Heute wurden 300 CHF für irgendetwas ausgegeben.

Der Benutzer ändert lediglich:

```
Kontostand: 8'120 CHF
```

Die gesamte Zukunftsberechnung wird automatisch neu berechnet.

---

# 3. Dashboard

Das Dashboard sollte die wichtigste Seite der gesamten Anwendung sein.

Keine Chartsammlung und keine komplizierten Statistiken.

Beispiel:

```
CASH PLANNER

Kontostand
CHF 8'120.00
[ Kontostand aktualisieren ]

──────────────────────────────

Reserviert
CHF 3'420.00

Frei verfügbar
CHF 4'700.00

──────────────────────────────

Nächste Zahlung

15. August
Internet
CHF 60.00

──────────────────────────────

Niedrigster erwarteter Kontostand
nächste 90 Tage

CHF 2'850.00
am 24. September

──────────────────────────────

NÄCHSTE ZAHLUNGEN

15.08   Internet                 -60
25.08   Lohn                  +7'000
01.09   Miete                 -2'000
05.09   Krankenkasse            -450
15.09   Internet                 -60
25.09   Lohn                  +7'000
```

---

# 4. Zwei unterschiedliche Kennzahlen

Die Anwendung sollte zwei Dinge bewusst voneinander unterscheiden.

## Kontostand-Projektion

Hier wird mathematisch berechnet, wie sich der Kontostand entwickelt.

Beispiel:

```
Heute                      8'120
Internet                    -60
                           ─────
15.08                      8'060

Lohn                      +7'000
                           ─────
25.08                     15'060

Miete                     -2'000
                           ─────
01.09                     13'060
```

Das beantwortet:

> Reicht mein Geld zu jedem Zeitpunkt?

---

## Reserviertes Geld

Zusätzlich können zukünftige grosse Rechnungen anteilig berücksichtigt werden.

Beispiel:

```
Autoversicherung
1'200 CHF
jährlich am 15. Februar
```

Die Anwendung kann daraus berechnen:

```
100 CHF / Monat Reserve
```

Dadurch ist dieses Geld gedanklich nicht mehr frei verfügbar.

Beispiel:

```
Kontostand                 8'120
benötigte Reserven        -3'420
                          ──────
tatsächlich frei          4'700
```

---

# 5. Arten von Einträgen

Es sollte möglichst wenige Typen geben.

## Wiederkehrende Einnahme

Beispiele:

* Lohn
* Nebeneinkommen
* regelmäßige Rückerstattung

Attribute:

```
Name
Betrag
Startdatum
Wiederholung
optional Enddatum
```

---

## Wiederkehrende Ausgabe

Beispiele:

* Miete
* Krankenkasse
* Internet
* Versicherungen
* Hosting
* Abonnements
* Steuern

Attribute:

```
Name
Betrag
Fälligkeit
Wiederholung
Reserve ja/nein
```

---

## Einmalige Zahlung

Beispiele:

* Rechnung
* Steuerrechnung
* Reparatur
* grössere Anschaffung

Attribute:

```
Name
Betrag
Fälligkeitsdatum
Reserve ja/nein
```

Nach der Fälligkeit kann der Eintrag automatisch archiviert werden.

---

# 6. Wiederholungsregeln

Für Version 1 reichen:

```
einmalig

wöchentlich
monatlich
alle 2 Monate
alle 3 Monate
alle 6 Monate
jährlich
```

Intern sollte das System trotzdem flexibel aufgebaut sein.

Beispiel:

```
recurrence_unit = MONTH
recurrence_interval = 1
```

oder:

```
recurrence_unit = YEAR
recurrence_interval = 1
```

Damit wären später beispielsweise auch folgende Regeln möglich:

```
alle 2 Jahre
alle 4 Monate
```

---

# 7. Projektion

Die Anwendung generiert aus den Regeln zukünftige Ereignisse.

Standard:

```
12 Monate
```

Optional auswählbar:

```
30 Tage
90 Tage
6 Monate
12 Monate
24 Monate
```

Beispiel:

Aus:

```
Miete
2'000 CHF
jeden 1. des Monats
```

entstehen intern:

```
01.09.2026   -2'000
01.10.2026   -2'000
01.11.2026   -2'000
...
```

Diese müssen nicht in der Datenbank gespeichert werden.

Sie können bei jeder Berechnung aus der Regel erzeugt werden.

---

# 8. Wichtigste Berechnung

Ausgangspunkt:

```
currentBalance
```

Danach werden alle zukünftigen Ereignisse chronologisch sortiert.

Pseudo-Code:

```
balance = currentBalance

events = generateEvents(today, today + 12 months)

sort(events by date)

for event in events:
    balance += event.amount

    save projected balance
```

Anschliessend können berechnet werden:

```
aktueller Kontostand

niedrigster Kontostand

Datum des niedrigsten Kontostands

nächste Zahlung

Summe Fixkosten pro Monat

Summe Fixkosten pro Jahr
```

---

# 9. Unterdeckungswarnung

Sehr wichtig wäre eine einfache Warnung.

Beispiel:

```
⚠ Unterdeckung erwartet

Am 15. Februar 2027 würde dein
Kontostand auf

-620 CHF

fallen.

Fehlender Betrag:
620 CHF
```

Oder:

```
✓ Alle bekannten Verpflichtungen
  der nächsten 12 Monate sind gedeckt.
```

Das ist wahrscheinlich eine der wertvollsten Funktionen der gesamten App.

---

# 10. Reserve-System

Das Reserve-System sollte optional sein.

Ein Eintrag kann:

```
Reserve: AUS
```

oder

```
Reserve: AN
```

haben.

Beispiel:

```
Autoversicherung
CHF 1'200
jährlich
Reserve: AN
```

Das System kann den benötigten Betrag bis zur nächsten Fälligkeit berechnen.

Dadurch lässt sich eine Kennzahl anzeigen:

```
aktueller Kontostand        8'120
reservierter Betrag        3'420
                           ─────
frei verfügbar             4'700
```

Wichtig:

"Reserviert" bedeutet nur eine rechnerische Reserve.

Es findet keine echte Umbuchung statt.

---

# 11. Manuelle Kontostand-Aktualisierung

Die Aktualisierung sollte extrem schnell sein.

Auf dem Dashboard:

```
Aktueller Kontostand

[ 8'120.50 ]

[ Aktualisieren ]
```

Optional:

```
zuletzt aktualisiert:
07.08.2026 10:24
```

Mehr braucht es nicht.

---

# 12. Architektur

Für diese Anwendung würde ich bewusst eine sehr einfache Architektur wählen:

```
┌──────────────────────────────┐
│          Browser             │
│                              │
│ Desktop / Mobile / PWA       │
└──────────────┬───────────────┘
               │ HTTPS
               ▼
┌──────────────────────────────┐
│        Cash Planner          │
│                              │
│  Web UI                      │
│  REST API                    │
│  Business Logic              │
│                              │
│       ein Container          │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│          SQLite              │
│                              │
│ /data/cashplanner.db         │
└──────────────────────────────┘
```

Kein:

* Redis
* RabbitMQ
* PostgreSQL
* Elasticsearch
* Worker
* Microservices
* Message Queue

Für diese Anwendung wäre das alles unnötige Komplexität.

---

# 13. Empfohlener Tech-Stack

## Backend

Go

Warum:

* ein einzelnes Binary
* sehr kleines Docker-Image
* schnell
* praktisch keine Runtime-Abhängigkeiten
* sehr einfach zu deployen
* langfristig wartbar
* hervorragend für eine kleine Self-Hosted-Anwendung

---

## Datenbank

SQLite

Datei:

```
/data/cashplanner.db
```

Vorteile:

* keine separate Datenbank
* Backup = Datei sichern
* sehr zuverlässig
* mehr als ausreichend für einen Benutzer
* keine Datenbank-Konfiguration notwendig

SQLite WAL sollte aktiviert werden.

---

## Frontend

Ich würde eine von zwei Varianten wählen.

### Variante A – maximal simpel

Go Templates + HTMX

Vorteile:

* nur ein Projekt
* kein separates Frontend
* praktisch kein JavaScript-Build-System
* extrem kleines Image
* sehr wartbar

Meine Empfehlung für dieses Projekt.

### Variante B – moderner

Svelte / SvelteKit Frontend

plus

Go REST API

Vorteile:

* angenehmere interaktive Oberfläche
* PWA einfacher
* modernere UX

Nachteil:

* Node Build Pipeline
* mehr Abhängigkeiten

Für den tatsächlichen Funktionsumfang ist HTMX wahrscheinlich vollkommen ausreichend.

---

# 14. Projektstruktur

Beispielsweise:

```
cashplanner/
│
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── database/
│   ├── expenses/
│   ├── income/
│   ├── recurrence/
│   ├── projection/
│   ├── reserves/
│   └── web/
│
├── web/
│   ├── templates/
│   ├── static/
│   └── icons/
│
├── migrations/
│
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

---

# 15. Datenmodell

Eine einfache Tabelle reicht für fast alle finanziellen Einträge.

## entries

```
id
name
type
amount
start_date
recurrence_unit
recurrence_interval
reserve_enabled
active
notes
created_at
updated_at
```

type:

```
income
expense
```

recurrence_unit:

```
once
week
month
year
```

Beispiel:

```
id: 1
name: "Miete"
type: expense
amount: 2000
start_date: 2026-09-01
recurrence_unit: month
recurrence_interval: 1
reserve_enabled: false
```

---

## settings

```
key
value
```

Beispielsweise:

```
currency = CHF
projection_months = 12
```

---

## balance

Entweder nur:

```
current_balance
updated_at
```

oder besser eine kleine Historie:

```
id
balance
created_at
```

Dann sieht man später zumindest:

```
01.08   8'540 CHF
07.08   8'120 CHF
```

ohne einzelne Transaktionen kennen zu müssen.

Das würde ich bevorzugen.

---

# 16. Keine Floats für Geld

Geldbeträge sollten intern niemals als Float gespeichert werden.

Statt:

```
1234.50
```

wird gespeichert:

```
123450
```

also Rappen/Cents als Integer.

Beispiel:

```
CHF 1'200.50
```

=

```
120050
```

Damit entstehen keine Rundungsfehler.

---

# 17. API

Obwohl die erste UI serverseitig gerendert werden kann, würde ich intern eine saubere API vorsehen.

Beispiele:

```
GET    /api/entries
POST   /api/entries
GET    /api/entries/:id
PUT    /api/entries/:id
DELETE /api/entries/:id

GET    /api/balance
PUT    /api/balance

GET    /api/projection

GET    /api/dashboard
```

Dadurch könnte später problemlos eine:

* Mobile App
* CLI
* Home Assistant Integration
* iOS Shortcut
* externe Integration

darauf zugreifen.

---

# 18. Docker-Image

Ziel:

```
ghcr.io/<username>/cashplanner:latest
```

und versioniert:

```
ghcr.io/<username>/cashplanner:1.0.0
ghcr.io/<username>/cashplanner:1.1.0
ghcr.io/<username>/cashplanner:1.2.0
```

Das Image sollte alles enthalten.

Nur `/data` muss persistent sein.

---

# 19. Multi-Stage Docker Build

Prinzip:

```
Stage 1
Go-Anwendung kompilieren

        ↓

Stage 2
minimales Runtime-Image
```

Dadurch kann das finale Image sehr klein bleiben.

Beispielsweise:

```
golang:...
     ↓
  build
     ↓
distroless/alpine
```

Ich würde eher ein minimalistisches Debian/Distroless-Image verwenden, solange keine Shell im Container benötigt wird.

---

# 20. Docker Compose

Das gewünschte Deployment sollte am Ende ungefähr so simpel sein:

```
services:

  cashplanner:
    image: ghcr.io/example/cashplanner:latest
    container_name: cashplanner

    restart: unless-stopped

    ports:
      - "8080:8080"

    volumes:
      - ./data:/data

    environment:
      - TZ=Europe/Zurich
```

Dann:

```
docker compose up -d
```

Fertig.

---

# 21. Reverse Proxy

Auf deinem Server würde ich die Anwendung nicht direkt öffentlich auf Port 8080 freigeben.

Stattdessen:

```
Internet
   │
   ▼
Reverse Proxy
   │
   ├── TLS / HTTPS
   │
   ▼
Cash Planner
   │
   ▼
SQLite
```

Geeignet wären beispielsweise bestehende Installationen von:

* Traefik
* Caddy
* Nginx Proxy Manager
* nginx

Die Anwendung selbst muss davon nichts wissen.

---

# 22. Authentifizierung

Für Version 1 würde ich keine komplexe Benutzerverwaltung bauen.

Es gibt zwei sinnvolle Varianten.

## Variante 1

Authentifizierung komplett vom Reverse Proxy übernehmen lassen.

Beispielsweise:

```
Authentik
    ↓
Reverse Proxy
    ↓
Cash Planner
```

Das wäre für eine private Self-Hosted-App meine bevorzugte Lösung.

## Variante 2

Einfacher Benutzer + Passwort direkt in Cash Planner.

Aber:

* keine Registrierungen
* keine Rollen
* keine Teams
* keine Passwort-Reset-E-Mails

Nur:

```
username
password hash
```

---

# 23. Backup

Da alles in SQLite liegt, ist Backup extrem simpel.

Zu sichern:

```
./data/
```

Beispielsweise:

```
data/
  cashplanner.db
```

Optional könnte die Anwendung selbst einen Button anbieten:

```
Einstellungen
→ Backup herunterladen
```

und:

```
Backup wiederherstellen
```

---

# 24. Datenexport

Sehr wichtig, damit die Anwendung kein Lock-in erzeugt.

Mindestens:

```
Export JSON
Export CSV
```

JSON könnte alle Daten vollständig enthalten.

Beispiel:

```
{
  "balance": 812050,
  "currency": "CHF",
  "entries": [...]
}
```

Damit kann selbst bei einem kompletten Projektabbruch alles problemlos wiederhergestellt werden.

---

# 25. Updates

GitHub Repository:

```
main
  ↓
GitHub Actions
  ↓
Docker Build
  ↓
GHCR
  ↓
Server
```

Tags:

```
v1.0.0
v1.1.0
v1.2.0
```

Auf dem Server:

```
docker compose pull
docker compose up -d
```

Die Anwendung führt beim Start automatisch notwendige Datenbank-Migrationen durch.

---

# 26. Releases

Ich würde Semantic Versioning verwenden:

```
1.0.0

MAJOR.MINOR.PATCH
```

Beispiele:

```
1.0.1
Bugfix

1.1.0
neue Funktion

2.0.0
inkompatible Änderung
```

---

# 27. Healthcheck

Die Anwendung sollte bereitstellen:

```
GET /health
```

Antwort:

```
200 OK
```

Dadurch funktioniert beispielsweise:

```
healthcheck:
  test: ...
```

und Docker kann erkennen, ob die Anwendung sauber läuft.

---

# 28. Zeitzone

Zeitberechnungen sind bei dieser Anwendung wichtig.

Default:

```
Europe/Zurich
```

Ein Zahlungstag sollte als lokales Datum behandelt werden:

```
2027-02-15
```

nicht als:

```
irgendein UTC-Zeitpunkt
```

Für Rechnungen interessiert primär das Datum und nicht die Uhrzeit.

---

# 29. Währung

Version 1 sollte bewusst nur eine Hauptwährung verwenden.

Beispielsweise:

```
CHF
```

Keine:

* Wechselkurse
* Crypto
* Multi-Currency-Buchhaltung

Später könnte man das erweitern, aber für den ursprünglichen Zweck bringt es keinen Mehrwert.

---

# 30. UI-Seiten

Ich würde die gesamte Anwendung auf fünf Seiten begrenzen.

## 1. Dashboard

```
Kontostand
reserviert
frei verfügbar
nächste Rechnung
Warnungen
nächste Zahlungen
```

## 2. Fixkosten

Liste:

```
Miete
Krankenkasse
Internet
Versicherungen
...
```

## 3. Einnahmen

Liste:

```
Lohn
weitere regelmäßige Einnahmen
```

## 4. Timeline

Chronologische Ansicht:

```
August
September
Oktober
November
...
```

## 5. Einstellungen

```
Währung
Projektionszeitraum
Backup
Restore
Export
```

Mehr würde ich am Anfang nicht bauen.

---

# 31. Mobile First

Da man den Kontostand vermutlich häufig schnell vom Handy aktualisieren möchte, sollte die UI primär für Mobile gebaut werden.

Aufruf:

```
cash.example.ch
```

Dann:

```
Kontostand
[ 8120.50 ]

Speichern
```

Das sollte innerhalb weniger Sekunden möglich sein.

Optional kann die Website als PWA installierbar sein.

Dann verhält sie sich fast wie eine normale Handy-App.

---

# 32. Was ausdrücklich NICHT in Version 1 gehört

Kein:

```
Bank Sync

SIX API

LUKB API

Kreditkarten Sync

CSV Import von Banken

automatische Kategorisierung

Machine Learning

Belegscanner

OCR

Anlageverwaltung

Aktien

Crypto

offene Rechnungsverwaltung

doppelte Buchhaltung

mehrere Benutzer

Familienaccounts

Push Notifications

komplizierte Charts

AI Assistent
```

Das alles würde den eigentlichen Vorteil der Anwendung zerstören:

> Sie soll lächerlich einfach sein.

---

# 33. MVP

Version 0.1 sollte nur Folgendes können:

### Einträge

* Einnahme hinzufügen
* Ausgabe hinzufügen
* monatlich
* jährlich
* einmalig
* bearbeiten
* löschen

### Kontostand

* aktuellen Kontostand setzen

### Berechnung

* zukünftige Ereignisse generieren
* Kontostand simulieren
* niedrigsten zukünftigen Kontostand bestimmen
* Unterdeckung erkennen

### Dashboard

* aktueller Kontostand
* nächste Zahlung
* nächste 10 Ereignisse
* niedrigster erwarteter Kontostand

### Infrastruktur

* SQLite
* Dockerfile
* Docker Compose
* persistentes Volume

Das reicht bereits für eine tatsächlich nutzbare Anwendung.

---

# 34. Version 0.2

Danach:

* Reserve-System
* Frei-verfügbar-Betrag
* Timeline
* 30/90/365-Tage-Projektionen
* Backup/Restore
* JSON Export

---

# 35. Version 0.3

Erst danach:

* PWA
* Dark Mode
* Auth
* API Tokens
* CSV Export
* optionale Benachrichtigungen

---

# 36. Spätere optionale Erweiterung: Bankintegration

Die Architektur sollte Bankintegration nicht voraussetzen.

Aber sie sollte später möglich sein.

Aktuell:

```
Benutzer
   │
   │ manueller Kontostand
   ▼
Cash Planner
```

Später theoretisch:

```
Bank Provider
   │
   │ Balance API
   ▼
Cash Planner
```

Die einzige Information, die Cash Planner wirklich benötigt, wäre:

```
aktueller Kontostand
```

Es müssten deshalb selbst bei einer späteren Bankintegration nicht zwingend sämtliche Transaktionen importiert werden.

Das ist ein wichtiger Architekturvorteil.

---

# 37. Kernphilosophie

Die Anwendung sollte immer diesem Prinzip folgen:

> So wenig Daten wie möglich eingeben, um eine konkrete finanzielle Entscheidung treffen zu können.

Nicht:

> Dokumentiere dein gesamtes finanzielles Leben.

Der zentrale Wert der Anwendung ist daher nicht das Erfassen von Daten.

Der zentrale Wert ist:

```
Verpflichtungen
      +
aktueller Kontostand
      +
zukünftige Einnahmen
      ↓
Wie viel Geld darf ich ausgeben?
```

---

# 38. Technisches Zielbild

Am Ende sollte das komplette Deployment aus ungefähr diesen Dateien bestehen:

```
docker-compose.yml

data/
  cashplanner.db
```

Und der Betrieb sollte nur benötigen:

```
docker compose pull
docker compose up -d
```

Die Anwendung selbst liegt vollständig im eigenen Docker-Image.

Damit ist sie:

* einfach zu installieren
* einfach zu aktualisieren
* einfach zu sichern
* einfach umzuziehen
* unabhängig von externen Diensten
* vollständig selbst gehostet

## Empfohlene technische Kombination

```
Backend         Go
Frontend        Go Templates + HTMX
CSS             simples eigenes CSS
Datenbank       SQLite
Deployment      Docker
Image Registry  GitHub Container Registry
CI/CD           GitHub Actions
Reverse Proxy   bestehender Proxy
Auth            Proxy/Authentik oder später intern
Zeitzone        Europe/Zurich
Hauptwährung    CHF
```

Das ergibt eine sehr kleine, robuste und langfristig wartbare Anwendung ohne unnötigen technischen Ballast.
