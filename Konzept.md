# Konzept: Simple Cash Planner (EasyPlan)

## 1. Ziel

Eine extrem einfache self‑hosted Webanwendung, die ausschließlich dabei hilft, monatliche, wöchentliche und tägliche Fixkosten vollständig abzuschätzen. Es werden **keine Einnahmen und kein Kontostand erfasst**. Die App beantwortet jederzeit drei Fragen:

1. Wie viel Geld brauche ich pro Monat, Woche und Tag?
2. Wie viel Geld muss auf welchem Konto (z. B. Lifestyle, Sparen, Steuern) mindestens vorhanden sein, um die nächsten Zahlungen sicher zu begleichen?
3. Welche konkreten Zahlungen stehen als Nächstes an?

---

## 2. Grundprinzip

Der Benutzer pflegt ausschließlich **Ausgaben**. Jede Ausgabe wird einem Konto zugeordnet. Einnahmen und Kontostände werden bewusst nicht erfasst – die App berechnet rein aus den Verpflichtungen, wie viel Geld vorhanden sein muss.

Beispiel:

| Konto    | Name             |     Betrag | Wiederholung | Datum       |
| -------- | ---------------- | ---------: | ------------ | ----------- |
| Lifestyle| Miete            | -2'000 CHF | monatlich    | 1.          |
| Lifestyle| Krankenkasse     |   -450 CHF | monatlich    | 5.          |
| Lifestyle| Internet         |    -60 CHF | monatlich    | 15.         |
| Sparen   | Autoversicherung | -1'200 CHF | jährlich     | 15. Februar |
| Steuern  | Serafe           |   -335 CHF | jährlich     | 20. März    |

Das Vorzeichen wird bei der Anzeige gesetzt; intern werden ausschließlich **positive Beträge in Minor Units (Rappen)** gespeichert.

---

## 3. Dashboard

Das Dashboard zeigt pro Konto den monatlichen, wöchentlichen und täglichen Bedarf sowie die nächsten anstehenden Zahlungen.

```
EASYPLAN

Lifestyle
  Monat:  2'510 CHF   Woche:   585 CHF   Tag:   84 CHF
  Nächste:  Miete 01.09   -2'000 CHF

Sparen
  Monat:    100 CHF   Woche:    23 CHF   Tag:    3 CHF
  Nächste:  Autoversicherung 15.02.2027   -1'200 CHF

Steuern
  Monat:     28 CHF   Woche:     6 CHF   Tag:    1 CHF
  Nächste:  Serafe 20.03.2027   -335 CHF

Insgesamt
  Monat:  2'638 CHF   Woche:   614 CHF   Tag:   88 CHF
```

Darüber hinaus zeigt das Dashboard eine **30‑Tage‑Reserve‑Übersicht** pro Konto, siehe §8.

---

## 4. Konten

Konten dienen ausschließlich der Gruppierung von Ausgaben. Attribute:

- Name (z. B. „Lifestyle“, „Sparen“, „Steuern“)
- Optionale Beschreibung
- Sortier­reihenfolge für die Anzeige
- Aktiv/Inaktiv

Bei der Installation werden sinnvolle Standardkonten angelegt. Benutzer können weitere Konten hinzufügen oder deaktivieren.

---

## 5. Arten von Einträgen

### Wiederkehrende Ausgabe

```
Name
Konto
Betrag (positiv)
Startdatum
Wiederholung (siehe §6)
optional Enddatum
Notizen
```

### Einmalige Ausgabe

```
Name
Konto
Betrag (positiv)
Fälligkeitsdatum
Notizen
```

Nach der Fälligkeit wird der Eintrag automatisch deaktiviert (`is_active = false`), bleibt aber für den Verlauf sichtbar.

---

## 6. Wiederholungsregeln

Intern einheitlich:

```
recurrence_unit     = { day, week, month, year, once }
recurrence_interval = N (>0)
```

Version 1 unterstützt:

- monatlich (`month`, `1`)
- alle N Monate (`month`, `N`)
- jährlich (`year`, `1`)
- wöchentlich (`week`, `1`)
- einmalig (`once`)

Wiederholungen werden **deterministisch** aus `start_date` und `ends_on` berechnet; es werden keine einzelnen Termine persistiert. **Monatsende‑Regel:** Ist der gewählte Tag größer als der letzte Tag des Zielmonats, wird auf den letzten gültigen Tag des Monats verschoben (z. B. 31. → 28./29. Februar). Ein Schaltjahr wird korrekt behandelt.

---

## 7. Perioden‑Berechnung

### Monatsbedarf

```
monthly_equivalent(entry) =
    case recurrence_unit:
        month: amount_minor * interval
        year:  amount_minor * interval / 12
        week:  amount_minor * interval * 4,345
        once:  0  (wird im Fälligkeitsmonat addiert)

monatlicher Bedarf (Konto) = Σ monthly_equivalent(entry)  + Σ einmaliger Beträge im aktuellen Monat
```

### Wochenbedarf

```
weekly = monthly / 4,345
```

### Tagesbedarf

```
daily = monthly / 30,4375
```

Pro Konto und gesamt.

---

## 8. Benötigte Mindest‑Reserve pro Konto

Zusätzlich zum Durchschnitt wird der **Mindestbedarf bis zu einem wählbaren Zeitfenster** berechnet:

```
reserve_window_days = 30 (einstellbar: 7, 30, 60, 90)
reserve(account) = Σ amount_minor
                   für alle Einträge dieses Kontos,
                   deren nächste Fälligkeit <= today + reserve_window_days
```

Anzeige:

```
Lifestyle – Reserve 30 Tage: 2'510 CHF
Sparen    – Reserve 30 Tage:   100 CHF (nächste Fälligkeit erst in 180 Tagen)
```

Eine Warnung erscheint, sobald die Reserve einen vom Benutzer pro Konto hinterlegten **Wunsch‑Puffer** (optional, in `settings`) überschreitet.

---

## 9. Timeline

Chronologische Ansicht aller anstehenden Zahlungen, gruppiert nach Monat:

```
August
  15.   Internet        -60

September
  01.   Miete          -2'000
  05.   Krankenkasse     -450
  15.   Internet          -60

…
```

Die Timeline wird aus den Wiederholungsregeln für den gewählten Zeitraum (Standard 12 Monate) on‑the‑fly erzeugt.

---

## 10. Datenmodell

### accounts

```
id            uuid
name          text
description   text
sort_order    int
is_active     bool
created_at    timestamp
updated_at    timestamp
```

### entries

```
id                  uuid
name                text
account_id          uuid (FK accounts)
amount_minor        integer (>0)
start_date          date
recurrence_unit     text (day|week|month|year|once)
recurrence_interval integer (>0, default 1)
ends_on             date null
notes               text
is_active           bool
created_at          timestamp
updated_at          timestamp
```

### settings

```
key   text primary key
value text
```

Beispiel:

```
projection_days        = 30
currency               = CHF
timezone               = Europe/Zurich
default_reserve_window = 30
```

Geldbeträge werden ausschließlich als **Minor Units (Rappen)** als `integer` gespeichert. `amount_minor` ist immer positiv; das Vorzeichen ergibt sich aus dem Eintragstyp (immer `expense`).

---

## 11. Service‑Schicht

Die Berechnungslogik liegt in `internal/service` und ist rein (keine I/O‑Abhängigkeiten), deterministisch und vollständig testbar:

- `MonthlyEquivalent(entry) int`
- `PeriodSummary(accountID string) (month, week, day int)`
- `ReserveRequired(accountID string, days int) int`
- `Timeline(from, to time.Time) []Event`

HTTP‑Handler rufen diese Funktionen auf; spätere Schnittstellen (CLI, REST, Mobile) können dieselben Funktionen wiederverwenden.

---

## 12. Docker (Produktion und Entwicklung)

### Produktion

Multi‑Stage `Dockerfile`:

```
Stage 1: golang:1.22 → go build -o /out/cashplanner ./cmd/server
Stage 2: gcr.io/distroless/static-debian12
         COPY /out/cashplanner /cashplanner
         USER nonroot
         EXPOSE 8080
         HEALTHCHECK --interval=30s --timeout=3s CMD ["/cashplanner", "healthcheck"]
```

Ergebnis: kleines, statisch gelinktes Binary, Image < 20 MB. Persistiert wird ausschließlich `/data` (SQLite).

`docker-compose.yml`:

```yaml
services:
  cashplanner:
    image: ghcr.io/<owner>/cashplanner:1.0.0
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=Europe/Zurich
```

Reverse Proxy (Caddy, Traefik, Nginx) übernimmt TLS und optional Authentifizierung.

### Entwicklung

`docker-compose.dev.yml`:

```yaml
services:
  cashplanner-dev:
    image: golang:1.22
    working_dir: /app
    volumes:
      - .:/app
      - cashplanner-dev-data:/data
    ports:
      - "127.0.0.1:8080:8080"
    command: ["air", "-c", ".air.toml"]
    environment:
      - TZ=Europe/Zurich
```

`air` (https://github.com/cosmtrek/air) wird als Dev‑Tool in `tools/air` versioniert. Änderungen am Go‑Code werden automatisch erkannt, kompiliert und der Server neu gestartet. SQLite‑Migrations laufen beim Start.

---

## 13. Sicherheit & Auth

- V1: keine eingebaute Authentifizierung. Der Standard‑Service bindet ausschließlich `127.0.0.1`, Sicherheit erfolgt über den vorgelagerten Reverse Proxy (Authentik, OAuth2‑Proxy, Caddy mit Basic Auth).
- Container läuft als `nonroot`, Root‑Dateisystem `read_only`, Schreibzugriff nur auf `/data` und temporäre Verzeichnisse.
- SQLite im WAL‑Modus, `PRAGMA foreign_keys=ON`.
- Tägliche Backups via Cron/systemd‑Timer (`sqlite3 cashplanner.db ".backup /backup/…"`) oder in‑app Backup‑Button.

---

## 14. Backup & Export

- **Backup**: UI‑Button ruft die SQLite‑Backup‑API (`VACUUM INTO '/data/backup/cashplanner-YYYYMMDD.db'`) auf und liefert eine konsistente Datei.
- **Restore**: Upload einer `.db`‑Datei, Validierung (SQLite‑Header, Migration‑Stand), Bestätigungs­dialog, anschließend atomarer Austausch.
- **Export**: JSON‑Export aller Konten, Einträge und Einstellungen; CSV‑Export der Einträge (`name,account,amount_minor,start_date,recurrence_unit,recurrence_interval,ends_on`).

---

## 15. Zeitzone & Währung

- Standard‑Zeitzone `Europe/Zurich`, einstellbar in `settings`.
- Fälligkeitstermine werden als lokale Datumswerte behandelt.
- Währung: V1 unterstützt genau eine Währung (Standard `CHF`). Keine Wechselkurse, keine Crypto.

---

## 16. UI‑Seiten

1. **Dashboard** – pro Konto Monats‑/Wochen‑/Tages­bedarf, nächste Zahlungen, Reserve­übersicht.
2. **Konten** – Liste und Bearbeitung der Konten.
3. **Einträge** – Liste aller wiederkehrenden und einmaligen Ausgaben, CRUD.
4. **Timeline** – chronologische Ansicht der nächsten 12 Monate.
5. **Einstellungen** – Zeitzone, Projektions­fenster, Backup, Export.

Das Layout ist Mobile‑first, eigenes CSS, HTMX für partielle Aktualisierungen.

---

## 17. Mobile First & PWA

- Layout primär für Smartphones optimiert.
- Ein‑Klick‑Bearbeitung einer Ausgabe möglich.
- PWA‑Manifest und Service Worker installierbar; sensible Daten werden **nicht** im Browser gecacht.

---

## 18. Was ausdrücklich NICHT in Version 1 gehört

- Einnahmen‑/Kontostands‑Tracking
- Bank‑APIs, Kreditkarten‑Sync, CSV‑Import von Banken
- Mehrere Benutzer, Teams, Rollen
- AI‑Assistent
- Komplexe Charts, doppelte Buchhaltung
- Multi‑Currency, Wechselkurse, Crypto

---

## 19. Benachrichtigungen

Die App soll den Benutzer aktiv über anstehende Ereignisse und kritische Zustände informieren, ohne dass er die Seite öffnen muss. Ziel ist ein einziger, klarer **Outbound‑Webhook**, der von beliebigen Empfängern konsumiert werden kann.

### 19.1 Architektur

```
Notificator
  ├─ Eventquelle   (z. B. „Eintrag fällig in 3 Tagen“, „Reserve überschritten“)
  ├─ Dispatcher    (formatiert JSON, retry, rate‑limit)
  └─ Targets       (Gotify, Discord, Telegram, ntfy.sh, WhatsApp‑Bridge, …)
```

Der Notificator ist ein eigenes Package `internal/notify`, das über ein Interface angebunden ist. Pro Event wird eine `Notification`-Struct erzeugt:

```
type Notification struct {
    Event     string    // "entry.due_soon"
    Account   string
    Title     string
    Body      string
    Amount    int64
    DueDate   time.Time
    Severity  string    // info | warning | critical
}
```

### 19.2 Auslöser (V1)

| Auslöser                  | Schwellwert (einstellbar) | Severity |
|---------------------------|---------------------------|----------|
| Eintrag in X Tagen fällig | `due_soon_days` (Default 3)| info     |
| Reserve überschritten     | pro Konto `wish_buffer`   | warning  |
| Wunsch‑Puffer stark überschritten | Faktor 1,5            | critical |
| Wöchentliche Zusammenfassung | Sonntag 18:00 lokal      | info     |

### 19.3 Versand‑Kanäle

Vorrangig wird **Gotify** (self‑hosted, klein, kostenlos) als Ziel unterstützt, da keine externen Drittanbieter benötigt werden. Zusätzlich werden gängige Webhook‑Ziele per einfachem HTTP‑POST unterstützt:

| Kanal      | URL‑Schema                              | Auth               |
|------------|-----------------------------------------|--------------------|
| Gotify     | `https://gotify.example.com/message?token=…` | Header `X-Gotify-Key` |
| ntfy.sh    | `https://ntfy.sh/<topic>`               | optional Basic Auth |
| Discord    | `https://discord.com/api/webhooks/<id>/<token>` | – |
| Telegram   | `https://api.telegram.org/bot<token>/sendMessage` | – |
| WhatsApp   | Bridge (z. B. `wasabi`, `whatsapp-web.js`) | Custom |
| Generic    | beliebiger Webhook                      | optional Header    |

### 19.4 Konfiguration

In `settings`:

```
notify_enabled        = true
notify_target_kind    = gotify   # gotify|ntfy|discord|telegram|whatsapp|generic
notify_target_url     = https://gotify.example.com/message?token=…
notify_target_token   = …
notify_due_soon_days  = 3
notify_weekly_summary = true
```

### 19.5 Sicherheit

- Webhook‑URL und Token werden ausschließlich serverseitig gespeichert (`/data`), nicht im Browser.
- TLS‑Pflicht für alle externen Ziele; HTTP wird abgelehnt.
- Retry mit exponentiellem Backoff (max 3 Versuche), Fehler werden geloggt, nicht an Benutzer gesendet.
- Rate‑Limit: maximal 1 Benachrichtigung pro Event und Tag.

### 19.6 UI

Einstellungen‑Seite bietet:

- An/Aus‑Schalter für Benachrichtigungen
- Dropdown Ziel‑Typ
- URL‑ und Token‑Eingabe (Passwort‑Feld)
- „Test‑Nachricht senden“‑Button
- Liste der letzten 20 gesendeten Benachrichtigungen (Status: ok/fehler)

### 19.7 Datenschutz

Benachrichtigungen enthalten **niemals** den aktuellen Kontostand oder andere vertrauliche Detaildaten, sondern nur aggregierte Beträge und Fälligkeitstage.

---

## 20. Sprint‑Plan

| Sprint | Ziel                              | Tasks |
|--------|-----------------------------------|-------|
| **0 – Setup** | Projektgerüst und Entwicklungsumgebung | 1. Repo‑Initialisierung, README, Konzept.<br>2. Multi‑Stage `Dockerfile`, `docker-compose.yml`, `docker-compose.dev.yml` mit `air`.<br>3. CI‑Pipeline (Build, Test, Lint) in GitHub Actions.<br>4. SQLite‑Migrations‑Skript (`accounts`, `entries`, `settings`). |
| **1 – Domäne** | Kernlogik für Ausgaben | 1. `accounts`‑Repository + Service + Tests.<br>2. `entries`‑Repository + Service + Tests.<br>3. Wiederholungs­generator (`recurrence_unit`/`interval`, Monatsende‑Regel, `ends_on`).<br>4. Edge‑Case‑Tests: 31. Januar → Februar, Schaltjahr, Enddatum. |
| **2 – Berechnung** | Perioden‑Berechnung & Reserve | 1. `MonthlyEquivalent`, `PeriodSummary` (Monat/Woche/Tag).<br>2. `ReserveRequired(account, days)`.<br>3. Aggregations‑Service pro Konto und gesamt.<br>4. Tests für Jahres‑/Quartals­beträge und mehrere Konten. |
| **3 – UI Grundgerüst** | Dashboard, Einträge, Konten | 1. Layout‑Basis (Mobile‑first, eigenes CSS).<br>2. Dashboard‑Ansicht mit Perioden­zahlen und Reserve­anzeige.<br>3. Konten‑ und Einträge‑Listen mit HTMX‑Formularen.<br>4. Einstellungs‑Seite (Zeitzone, Projektions­fenster, Wunsch‑Puffer). |
| **4 – Timeline & Warnungen** | Vorausschau & Sicherheit | 1. Timeline‑Service (`Timeline(from, to)`).<br>2. Timeline‑Ansicht (Monats­gruppierung).<br>3. Warnungs­logik, wenn `ReserveRequired` einen pro Konto hinterlegten Wunsch‑Puffer überschreitet. |
| **5 – Backup & Export** | Datenmigration | 1. SQLite‑Backup‑API (`VACUUM INTO`).<br>2. UI‑Button „Backup herunterladen“.<br>3. JSON‑Export, CSV‑Export.<br>4. Restore‑Funktion mit Validierung und Bestätigungs­dialog. |
| **6 – Hardening & Release** | Produktionsreife | 1. Distroless‑Image, Non‑Root, Read‑Only FS.<br>2. Healthcheck‑Endpoint `/health`.<br>3. Reverse‑Proxy‑Beispiele (Caddy, Traefik).<br>4. Doku: Installation, Update, Backup‑Strategie.<br>5. Erstes Release `v1.0.0`. |
| **7 – Benachrichtigungen** | Aktive Erinnerungen | 1. `internal/notify`‑Package mit `Notifier`‑Interface.<br>2. Targets: Gotify, ntfy.sh, Discord, Telegram, Generic Webhook.<br>3. Eventquellen: `entry.due_soon`, `reserve.exceeded`, `weekly.summary`.<br>4. Retry‑Strategie (exponentielles Backoff) + Rate‑Limit.<br>5. Einstellungs‑Seite mit Test‑Button und Log.<br>6. TLS‑Prüfung, keine Klartext‑Tokens im Browser.<br>7. Dokumentation: Gotify‑Setup, Discord‑Webhook, Telegram‑Bot. |
| **8 – Optional** | Komfort | 1. PWA‑Manifest, Offline‑Shell.<br>2. Dark Mode.<br>3. CSV‑Import von Ausgaben‑Listen.<br>4. API‑Tokens für externe Skripte. |

Jeder Sprint endet mit einem Review auf dem `master`‑Branch über einen **non‑fast‑forward Merge** (`git merge --no-ff`) gemäß `CODE_OF_CONDUCT`.