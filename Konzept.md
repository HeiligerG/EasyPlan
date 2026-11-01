# Konzept: EasyPlan

Self-hosted Mini-App, die Fixkosten zusammenrechnet. Leitsatz: **so wenig wie möglich bauen, so schnell wie möglich nutzbar.**

> **Hinweis:** Dieses Dokument ist das ursprüngliche Konzept und dient dem Gesamtverständnis. Verbindlich sind die Dokumentation unter `docs/` und der Code.

## 1. Ziel

Es werden nur **Ausgaben** erfasst, keine Einnahmen und keine Kontostände. Die App beantwortet drei Fragen:

1. Wie viel Geld brauche ich pro Monat, Woche und Tag?
2. Wie viel muss pro Konto (Lifestyle, Sparen, Steuern, …) in den nächsten N Tagen mindestens vorhanden sein?
3. Welche Zahlungen kommen als Nächstes?

## 2. Leitplanken (Einfachheit)

- **Ein Binary, eine SQLite-Datei, keine weiteren Dienste.**
- Go mit Standardbibliothek (`net/http`, `html/template`, `embed`), SQLite über `modernc.org/sqlite` (ohne CGO), HTMX und CSS als eingebettete Dateien. Kein Node, kein Build-Schritt fürs Frontend, kein ORM.
- Konfiguration über Umgebungsvariablen, **keine** settings-Tabelle, **keine** Einstellungsseite.
- Keine eigene Authentifizierung (siehe §9).
- Berechnungen werden bei Bedarf live gemacht, es werden keine abgeleiteten Daten gespeichert.

## 3. Datenmodell

```
accounts
  id          integer primary key
  name        text not null unique
  sort_order  integer not null default 0
  is_active   integer not null default 1

entries
  id          integer primary key
  account_id  integer not null references accounts(id)
  name        text not null
  amount_minor integer not null check (amount_minor > 0)   -- Rappen
  start_date  text not null                                -- YYYY-MM-DD
  unit        text not null check (unit in ('once','week','month','year'))
  interval    integer not null default 1 check (interval > 0)
  ends_on     text null                                    -- YYYY-MM-DD
  notes       text not null default ''
```

- Beträge sind immer positive Rappen (`int64`). Das Vorzeichen (Minus) entsteht erst bei der Anzeige.
- Datumswerte sind lokale Kalendertage ohne Uhrzeit. Die Zeitzone kommt aus der Umgebungsvariable `TZ`.
- Migrationen liegen als SQL-Dateien im Binary (`embed`) und laufen beim Start. Fremdschlüssel sind aktiv (`PRAGMA foreign_keys=ON`), SQLite läuft im WAL-Modus.
- Standardkonten beim ersten Start: Lifestyle, Sparen, Steuern. Konten lassen sich umbenennen, hinzufügen und deaktivieren (kein Löschen, solange Einträge existieren).

## 4. Wiederholungen

| unit | Bedeutung | Beispiel |
|---|---|---|
| `once` | einmalig am `start_date` | Zahnarzt 12.11. |
| `week` | alle `interval` Wochen | `week`, 2 = alle 2 Wochen |
| `month` | alle `interval` Monate | `month`, 3 = quartalsweise |
| `year` | alle `interval` Jahre | Versicherung jährlich |

Regeln:

- Der n-te Termin wird **immer direkt aus `start_date` berechnet** (`start_date + n * interval`), nie vom vorherigen Termin aus. So gilt: 31.1. → 28.2. → 31.3. (und nicht 28.3.).
- **Monatsende:** Gibt es den Tag im Zielmonat nicht, wird der letzte Tag des Monats genommen (Schaltjahr inklusive).
- Termine gelten ab `start_date` bis und mit `ends_on` (falls gesetzt).
- Es werden keine einzelnen Termine gespeichert. Ein einmaliger Eintrag in der Vergangenheit gilt in der Liste als „erledigt“. Dafür wird nichts in die DB geschrieben.

## 5. Berechnung

Alles leitet sich aus den **Kosten pro Jahr** ab, damit Monat, Woche und Tag zueinander passen. Ein Jahr hat 365,25 Tage.

```
Jahreskosten(Eintrag) =
    week:  amount * 365.25 / 7 / interval
    month: amount * 12 / interval
    year:  amount / interval
    once:  nicht enthalten

Monat = Jahreskosten / 12
Woche = Jahreskosten * 7 / 365.25
Tag   = Jahreskosten / 365.25
```

- Summiert wird pro Konto und gesamt, jeweils über die ungerundeten Werte. Gerundet wird **nur bei der Anzeige** auf ganze Franken.
- Dadurch kann die Summe der angezeigten Konto-Tageswerte um 1 CHF vom angezeigten Total abweichen. Das ist gewollt, es wird nicht nachkorrigiert.
- **Einmalige Ausgaben** sind kein laufender Bedarf. Sie fliessen nicht in Monat, Woche und Tag ein, erscheinen aber in Timeline und Reserve.

### Reserve

```
Reserve(Konto, N) = Summe aller Fälligkeiten des Kontos im Zeitraum [heute, heute + N Tage)
```

- Jede tatsächliche Fälligkeit zählt, auch wenn sie im Zeitraum mehrfach vorkommt.
- N ist 7, 30 (Standard), 60 oder 90 und wird auf dem Dashboard per Auswahl umgeschaltet (Query-Parameter, nicht gespeichert).
- Die Reserve nutzt dieselbe Terminberechnung wie die Timeline. Termine entstehen nur in `internal/plan`: `Occurrences` und `Next` teilen sich denselben Kern.

### Beispiel (Stand 1.10.2026)

| Konto | Eintrag | Betrag | Rhythmus |
|---|---|---|---|
| Lifestyle | Miete | 2'000 | monatlich, am 1. |
| Lifestyle | Krankenkasse | 450 | monatlich, am 5. |
| Lifestyle | Internet | 60 | monatlich, am 15. |
| Sparen | Autoversicherung | 1'200 | jährlich, 15.2. |
| Steuern | Serafe | 335 | jährlich, 20.3. |

| Konto | Monat | Woche | Tag | Reserve 30 Tage | Nächste Zahlung |
|---|---:|---:|---:|---:|---|
| Lifestyle | 2'510 | 577 | 82 | 2'510 | Miete 01.10. |
| Sparen | 100 | 23 | 3 | 0 | Autoversicherung 15.02.2027 |
| Steuern | 28 | 6 | 1 | 0 | Serafe 20.03.2027 |
| **Total** | **2'638** | **607** | **87** | **2'510** | |

„Nächste Zahlung“ eines Kontos ist der früheste Termin aller aktiven Einträge ab heute (heute inklusive, wie bei Reserve und Timeline); ab dem 2.10. wäre es „Krankenkasse 05.10.“ (siehe `docs/adr/0004-naechste-zahlung-pro-konto.md`).

Das Total der Tageswerte ist 87, die Summe der Zeilen 86. Das ist die erwähnte Rundungsabweichung.

## 6. Seiten

Mobile-first, ein CSS-File, HTMX für Formulare und Teil-Updates. Jede Seite funktioniert auch ohne JavaScript, soweit es sich ohne Aufwand machen lässt.

1. **Dashboard:** pro Konto und gesamt Monat/Woche/Tag, Reserve für N Tage, nächste Zahlung. Auswahl für N.
2. **Einträge:** Liste (gruppiert nach Konto) mit Anlegen, Bearbeiten, Löschen. Bearbeiten ist ein Klick auf die Zeile.
3. **Timeline:** die nächsten 12 Monate, nach Monat gruppiert, optional nach Konto filterbar.
4. **Konten:** anlegen, umbenennen, sortieren, deaktivieren.
5. **Backup:** kein eigener Seitenpunkt. Im Seitenfuss jeder Seite steht der Knopf „Backup herunterladen“. Er sendet `POST /backup` und liefert die Kopie der Datenbank als Download (siehe §8). Weitere Einstellungen gibt es nicht.

## 7. Code-Struktur

```
cmd/easyplan/main.go      Start, Konfiguration, Subcommand "healthcheck"
internal/store/           SQLite, Migrationen, Queries
internal/plan/            reine Berechnungslogik, ohne I/O
internal/web/             Handler, Templates, statische Dateien
```

`internal/plan` enthält:

- `Occurrences(entry, from, to) []time.Time`: alle Termine des Eintrags im Bereich `[from, to)`
- `Next(entry, from) (time.Time, bool)`: erster Termin des Eintrags ab `from`, auch bei langen Intervallen
- `Totals(entries) Need`: Perioden-Bedarf als `Need{Month, Week, Day}`, ungerundet in Rappen
- `ActiveOn(entries, today) []Entry`: nur die Einträge, die am Stichtag noch laufen
- `IsDone(entry, today) bool`: ein einmaliger Eintrag in der Vergangenheit gilt als erledigt
- `RoundCHF(minor float64) int64`: Rappen auf ganze Franken, kaufmännisch gerundet
- `Entry.Validate() error`: prüft Betrag, Intervall, Einheit und Datum
- `Reserve(entries, today, days) int64`
- `Timeline(entries, from, to) []Event`

`Occurrences` und `Next` teilen sich den Kern `nth`; nur dort entstehen Termine.

Diese Funktionen sind rein und deterministisch (`today` wird übergeben, keine `time.Now()`-Aufrufe darin). Dort liegen die Tests, vor allem für:

- Monatsende (31.1. → Februar, 29.2. im Schaltjahr)
- `ends_on` am Termin selbst und davor/danach
- Intervalle > 1 (Quartal, alle 2 Wochen)
- Summen über mehrere Konten und Rundung

Für Store und Handler genügt je ein Smoke-Test mit SQLite im Speicher.

## 8. Betrieb

**Konfiguration (Umgebungsvariablen):**

| Variable | Standard ausserhalb von Docker | Wert im Image | Bedeutung |
|---|---|---|---|
| `EASYPLAN_ADDR` | `127.0.0.1:8080` | `:8080` | Die Adresse, auf der der Server hört |
| `EASYPLAN_DATA` | `/data` | `/data` | Ordner der SQLite-Datei |
| `TZ` | `Europe/Zurich` | `Europe/Zurich` | Zeitzone |

Ausserhalb des Containers hört der Server nur auf dem Loopback, weil die App keine eigene Anmeldung hat. Im Image hört er auf allen Schnittstellen, damit der gemappte Port ankommt; dort beschränkt die Portbindung der Compose-Datei den Zugriff auf den Loopback des Hosts.

Die Währung ist fest CHF.

**Docker:** Multi-Stage-Dockerfile (`golang:1.27.3-bookworm` → `gcr.io/distroless/static-debian12:nonroot`), Nutzer `nonroot` (UID 65532), Healthcheck über `easyplan healthcheck` gegen `/health`. Persistiert wird nur `/data`.

```yaml
services:
  easyplan:
    image: easyplan:local
    build: .
    restart: unless-stopped
    read_only: true
    stop_grace_period: 15s
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=Europe/Zurich
```

**Entwicklung:** `go run ./cmd/easyplan` (Daten in `./data`). Kein Hot-Reload-Tool, kein Dev-Compose.

**Healthcheck:** `GET /health` antwortet mit dem Status 200 und dem Text `ok`, wenn die Datenbank erreichbar ist, sonst mit dem Status 503 und dem Text `unhealthy`. Das Image steuert den Check mit `HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3`. Drei Timeouts stecken in der Prüfung: Die Datenbank hat 2 s, der Client 3 s, die Container-Runtime 5 s.

**Backup:** Der Knopf im Seitenfuss jeder Seite sendet `POST /backup` und liefert die Kopie der Datenbank als Download. Der Server schreibt die Kopie mit `VACUUM INTO` in einen temporären Ordner im Datenordner, weil das Root-Dateisystem des Containers unbeschreibbar ist (`read_only: true`): Der Ordner hat die Rechte 0700, die Kopie die Rechte 0600. Im normalen Weg wird der Ordner nach dem Download gelöscht; bleibt er nach einem harten Abbruch liegen, entfernt die Anwendung ihn beim nächsten Start.

**Restore** läuft von Hand: Container stoppen, `easyplan.db-wal` und `easyplan.db-shm` des alten Standes entfernen, die Backup-Datei mit `sudo install -o 65532 -g 65532 -m 600 …` als `/data/easyplan.db` einsetzen, Container starten. Das steht in der README.

**Automatische Backups:** Für regelmässige Kopien reicht ein Cron-Job mit `sqlite3 easyplan.db ".backup …"`. Er braucht Schreibzugriff auf den Datenordner, denn die gemeinsame Speicherdatei `easyplan.db-shm` gehört dem Nutzer 65532 des Containers. Wer keinen Zugriff auf den Ordner braucht, holt die Kopie über die Adresse: `curl -X POST -H "Origin: http://127.0.0.1:8080" http://127.0.0.1:8080/backup -o easyplan-$(date +%Y%m%d-%H%M%S).db`. Der `Origin`-Header muss zum `Host` der Anfrage passen, sonst antwortet der Server mit 403.

## 9. Sicherheit

- Keine eingebaute Authentifizierung. Ausserhalb des Containers lauscht der Dienst nur auf `127.0.0.1`, im Image auf allen Schnittstellen des Containers, dort beschränkt die Portbindung der Compose-Datei den Zugriff auf den Loopback des Hosts. Zugriff und TLS regelt ein Reverse Proxy (Caddy, Traefik, Authentik, …). Die README enthält ein Caddy-Beispiel mit Basic Auth.
- Container als `nonroot` (UID 65532), Root-Dateisystem schreibgeschützt, Schreibzugriff nur auf `/data`, `no-new-privileges` und `cap_drop: ALL`.
- CSRF-Schutz ohne Token: Der Server prüft bei jedem schreibenden Request (POST/PUT/DELETE), dass der `Origin`-Header zum `Host` passt (HTMX sendet ihn mit). Eingaben werden serverseitig validiert (Betrag > 0, gültiges Datum, bekanntes `unit`).
- Der Reverse Proxy muss den `Host`-Header weitergeben: Der Vergleich ist immer gegen den `Host` der Anfrage, `X-Forwarded-Host` wird nicht ausgewertet. Caddy und Traefik setzen `Host` von Haus aus weiter, nginx braucht `proxy_set_header Host $host;`. Ohne erhaltenen `Host` sind alle Schreibzugriffe mit 403 abgelehnt, und die App sieht nur noch lesen.

## 10. Nicht in V1

Einnahmen und Kontostände, Bank-Anbindung, mehrere Benutzer, mehrere Währungen, Charts, Einstellungsseite, Wunsch-Puffer und Warnungen, Upload-Restore, JSON/CSV-Export, PWA, Dark Mode, Benachrichtigungen.

## 11. Später (nur bei Bedarf)

Reihenfolge nach Nutzen. Jeder Punkt ist klein und unabhängig.

1. **Benachrichtigung:** eine einzige Umgebungsvariable `EASYPLAN_NOTIFY_URL` (https, z. B. ntfy oder Gotify). Ein täglicher Check sendet „Zahlung in ≤ 3 Tagen“ und sonntags eine Wochenzusammenfassung. Kein Retry-System, kein Log, keine UI; Fehler gehen ins Server-Log. Keine Kontostände oder Notizen in der Nachricht.
2. **CSV-Export** der Einträge.
3. **Dark Mode** per `prefers-color-scheme` (nur CSS).
4. **PWA-Manifest** (installierbar, ohne Daten-Cache).

## 12. Umsetzungsplan

Ein Feature-Branch pro Phase, Review, dann Merge per GitHub-Pull-Request.

| Phase | Ergebnis | Inhalt |
|---|---|---|
| **1 – Kern** | Berechnung steht und ist getestet | Repo-Gerüst (`go.mod`, Ordner, `.gitignore`), `internal/plan` mit Tests, Migration und `internal/store` |
| **2 – UI** | App ist benutzbar | Layout und CSS, Dashboard, Einträge, Konten, Timeline |
| **3 – Betrieb** | Läuft im Container | Dockerfile, Compose, `/health` mit Docker-HEALTHCHECK, Backup-Knopf im Seitenfuss (`POST /backup`), README (Installation, Update, Backup, Restore, Caddy- und nginx-Beispiel), GitHub-Actions-Workflow mit den Jobs `go`, `vulncheck`, `docs` und `docker`, Dependabot-Konfiguration, Release `v1.0.0` (noch offen) |
| **4 – Optional** | nach Bedarf | Punkte aus §11 |

Nach Phase 2 lässt sich die App bereits im Alltag testen. Phase 3 ist nur nötig, sobald sie dauerhaft laufen soll.
