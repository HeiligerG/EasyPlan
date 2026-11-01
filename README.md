# EasyPlan

EasyPlan ist eine self-hosted Mini-App in Go, die Fixkosten zusammenrechnet. Sie beantwortet drei Fragen:

1. Wie viel Geld brauche ich pro Monat, Woche und Tag?
2. Wie viel muss pro Konto in den nächsten N Tagen mindestens vorhanden sein?
3. Welche Zahlungen kommen als Nächstes?

Erfasst werden nur Ausgaben. Es gibt keine Einnahmen, keine Kontostände und keine Bank-Anbindung. Die Währung ist fest CHF.

## Status

**Phase 3 von 3 ist abgeschlossen: Die App ist benutzbar und der Betrieb ist beschrieben.**

| Bereich | Stand |
|---|---|
| Berechnung (`internal/plan`) | fertig, getestet |
| Datenspeicher (`internal/store`) | fertig, getestet |
| Benutzeroberfläche (`internal/web`) | fertig, getestet |
| Server und Konfiguration (`cmd/easyplan`) | fertig, getestet |
| Betrieb, Backup, Installation | fertig, getestet |

## Leitplanken

- Ein Binary, eine SQLite-Datei, keine weiteren Dienste.
- Go mit Standardbibliothek, SQLite über `modernc.org/sqlite` ohne CGO, HTMX und CSS als eingebettete Dateien. Kein Node, kein Build-Schritt fürs Frontend, kein ORM.
- Konfiguration über Umgebungsvariablen, keine Einstellungsseite.
- Keine eigene Authentifizierung, der Zugriff regelt ein Reverse Proxy.
- Berechnungen entstehen bei Bedarf live, es werden keine abgeleiteten Daten gespeichert.

## Funktionen

Die Oberfläche besteht aus vier Seiten:

- **Dashboard** (`/`): Bedarf pro Monat, Woche und Tag, Reserve für die nächsten 7, 30, 60 oder 90 Tage und die nächste Zahlung, je Konto und als Total.
- **Einträge** (`/entries`): alle Einträge, nach Konto gruppiert. Anlegen, bearbeiten mit einem Klick auf die Zeile und löschen mit einer Bestätigungsseite.
- **Timeline** (`/timeline`): die Zahlungen der nächsten zwölf Monate, nach Monat gruppiert, mit der Summe je Monat und einem Filter auf das Konto.
- **Konten** (`/accounts`): anlegen, umbenennen, in der Reihenfolge verschieben, aktivieren und deaktivieren.

Alle Seiten funktionieren ohne JavaScript. Die Beträge stehen in ganzen Franken, die Daten im Format `TT.MM.JJJJ`. Im Seitenfuss jeder Seite steht der Knopf „Backup herunterladen“, er lädt die Datenbankdatei herunter.

## Installation

Voraussetzung ist Docker mit Compose. Git wird für das Update gebraucht.

```bash
git clone https://github.com/HeiligerG/EasyPlan.git
cd EasyPlan

# Der Datenordner gehört dem Nutzer 65532 (nonroot), den der Container
# im Image verwendet. Ohne dieses Recht kann der Container die Datenbank
# nicht anlegen.
mkdir -p ./data
sudo chown 65532:65532 ./data

docker compose up -d
```

`docker compose up -d` baut das Image, startet den Dienst im Hintergrund und kehrt sofort zurück. Die Daten liegen danach in `./data` und bleiben dort.

### Die Prüfung

```bash
curl -fsS http://127.0.0.1:8080/health
```

```text
ok
```

Die Antwort ist `ok` mit dem Status 200, wenn die Datenbank erreichbar ist. Ist sie es nicht, antwortet der Dienst mit `unhealthy` und dem Status 503.

Der Zustand des Containers:

```bash
docker compose ps
```

```text
NAME       IMAGE            STATUS                PORTS
easyplan-1 easyplan:local   Up 2 minutes (healthy) 127.0.0.1:8080->8080/tcp
```

In der Spalte `STATUS` steht `healthy`, sobald der Healthcheck des Containers erfolgreich war. Er läuft alle dreissig Sekunden und wird von Docker mit `--interval=30s --timeout=5s --start-period=10s --retries=3` gesteuert.

### Die Oberfläche

Im Browser ist die App danach unter `http://127.0.0.1:8080` erreichbar. Die Adresse hört nur auf dem Loopback, von aussen braucht es einen Reverse Proxy, siehe unten.

### Das Update

```bash
git pull
docker compose build
docker compose up -d
```

Die Daten bleiben im Bind-Mount `./data`. Das Update ersetzt nur das Programm. Offene Migrationen laufen beim Start des neuen Containers von selbst.

## Backup und Restore

### Das Backup

Im Seitenfuss jeder Seite steht der Knopf „Backup herunterladen“. Er liefert eine Kopie der SQLite-Datei, im Format `easyplan-JJJJMMTT-HHMMSS.db`.

Die Kopie entsteht im Server mit `VACUUM INTO`, in einem temporären Ordner im Datenordner. Das ist nötig, weil das Root-Dateisystem des Containers unbeschreibbar ist. Der temporäre Ordner hat die Rechte 0700, die Kopie die Rechte 0600, und beides wird nach dem Download wieder gelöscht. Bleibt der Ordner nach einem harten Absturz liegen, entfernt die Anwendung ihn beim nächsten Start.

Die Adresse antwortet nur auf `POST`:

```text
POST /backup
```

Ein `GET /backup` gibt es nicht. Ein `GET` muss nach RFC 9110 §9.2.1 sicher sein, ein Backup ist es nicht: Es kopiert die ganze Datenbank und erzeugt Last, ein Link, ein Prefetch des Browsers oder ein Suchroboter könnte es auslösen. Weil `POST` eine schreibende Methode ist, greift zusätzlich die Prüfung der Herkunft: Der `Origin`-Header muss zum `Host` der Anfrage passen, sonst antwortet der Server mit 403.

### Das Restore

Das Restore läuft von Hand:

```bash
# 1. Container stoppen
docker compose stop

# 2. WAL- und SHM-Datei des alten Standes entfernen
sudo rm -f ./data/easyplan.db-wal ./data/easyplan.db-shm

# 3. Die Backup-Datei an den Platz der Datenbank legen, mit den Rechten
#    des Container-Nutzers 65532
sudo install -o 65532 -g 65532 -m 600 \
  ~/Downloads/easyplan-20261031-214500.db ./data/easyplan.db

# 4. Container starten
docker compose up -d
```

Schritt 2 ist nicht optional: Die Dateien `easyplan.db-wal` und `easyplan.db-shm` gehören zum alten Stand und würden sonst auf die neue Datei gespielt. Die Backup-Datei selbst braucht keine Begleitdateien, `VACUUM INTO` schreibt eine vollständige Datenbank in eine einzige Datei.

Schritt 3 ersetzt die Datei und setzt Besitzer, Gruppe und Rechte im selben Schritt. Die Datenbank liegt damit nie mit fremden Rechten da, auch nicht für einen Moment.

Danach prüfen:

```bash
docker compose ps
curl -fsS http://127.0.0.1:8080/health
```

## Der Reverse Proxy

EasyPlan hat **keine eigene Authentifizierung**. Der Zugriff gehört hinter einen Reverse Proxy, der TLS beendet und die Anmeldung regelt.

### Caddy

```caddyfile
plan.example.com {
    # Basic Auth, der Hash entsteht mit: caddy hash-password
    basic_auth {
        plan <bcrypt-hash>
    }

    reverse_proxy 127.0.0.1:8080
}
```

`plan.example.com` steht hier für die eigene Domain. Caddy setzt den `Host`-Header von Haus aus weiter, es ist nichts zu tun.

### nginx

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
}
```

Die Zeile `proxy_set_header Host $host;` ist nötig. Die Anwendung vergleicht bei jedem schreibenden Request den `Origin`-Header mit dem `Host`-Header der Anfrage. Ohne erhaltenen `Host` sind alle Schreibzugriffe mit 403 abgelehnt, und die App sieht nur noch lesen. `X-Forwarded-Host` wird nicht ausgewertet.

## Die Adresse, auf der der Server hört

| Umgebung | Wert von `EASYPLAN_ADDR` | Erreichbar von aussen |
|---|---|---|
| Direkt gestartet (`go run`, eigenes Binary) | `127.0.0.1:8080` | nein, nur über den Loopback |
| Im Container-Image | `:8080` | nein, die Compose-Datei bindet nur `127.0.0.1:8080:8080` auf dem Host |

Der Standardwert ausserhalb von Docker ist der Loopback, weil die App keine eigene Anmeldung hat. Im Image hört der Server auf allen Schnittstellen des Containers, damit der gemappte Port ankommt; wer den Container direkt mit `docker run` startet, sollte deshalb selbst eine Beschränkung setzen:

```bash
docker build -t easyplan:local .
docker run --rm -p 127.0.0.1:8080:8080 -v ./data:/data easyplan:local
```

Die Form `127.0.0.1:8080:8080` ist Pflicht. `docker run -p 8080:8080` ohne den Teil `127.0.0.1:` bindet den Port auf alle Schnittstellen des Hosts: Die App ist dann im LAN erreichbar, ohne Anmeldung und ohne TLS, und mit ihr auch der Download des Backups mit der ganzen Datenbank.

## Die Konfiguration

Die Konfiguration kommt aus der Umgebung. Es gibt keine Einstellungsseite und keine Konfigurationsdatei.

| Variable | Standard ausserhalb von Docker | Wert im Image | Bedeutung |
|---|---|---|---|
| `EASYPLAN_ADDR` | `127.0.0.1:8080` | `:8080` | Die Adresse, auf der der Server hört |
| `EASYPLAN_DATA` | `/data` | `/data` | Der Ordner mit der SQLite-Datei `easyplan.db` |
| `TZ` | `Europe/Zurich` | `Europe/Zurich` | Die Zeitzone der Anwendung |

Eine leere Variable zählt wie nicht gesetzt, dann greift der Standardwert. Eine unbekannte Zeitzone bricht den Start ab.

`EASYPLAN_DATA` ist auch ausserhalb von Docker `/data`. Für die Entwicklung setzt man deshalb `EASYPLAN_DATA=./data`, sonst braucht das Anlegen des Ordners Root-Rechte.

## Betrieb ohne Docker

Go 1.27.3 oder neuer ist nötig.

```bash
# Prüfungen
go vet ./...
go test ./...

# App starten, erreichbar unter http://127.0.0.1:8080
EASYPLAN_DATA=./data go run ./cmd/easyplan
```

## Hinweise für den Betrieb

- **Das Image.** Die Compose-Datei baut das Image lokal und nennt es `easyplan:local`. `docker compose pull` funktioniert dafür nicht, das Image gibt es in keiner Registry. Gebaut wird mit `docker compose build` oder mit `docker compose up -d --build`.
- **Das Beenden.** Die Compose-Datei gibt dem Container 15 Sekunden für das Stoppen (`stop_grace_period`). Das Binary braucht zehn Sekunden, um die laufenden Anfragen zu beenden. Ohne die Reserve würde der Container vorher sterben.
- **Das Log.** Das Container-Log rotiert: `json-file` mit `max-size: 10m` und `max-file: 3`.
- **Die Datenbank und ihre Begleitdateien.** Im Datenordner liegen `easyplan.db`, `easyplan.db-wal` und `easyplan.db-shm`. Die beiden letzten Dateien gehören zum WAL-Modus von SQLite und sind normal.
- **Kein Netzwerkdateisystem.** Der WAL-Modus braucht eine gemeinsame Speicherdatei, die auf NFS nicht zuverlässig ist. Der Datenordner gehört auf eine lokale Platte.
- **Unbeschreibbares Root-Dateisystem.** Der Container schreibt nur in `/data`. Alles andere ist read-only, das ist eine Absicht.
- **Die Zeitzone.** Die Daten der Zeitzonenbank sind im Binary eingebettet (`time/tzdata`). `TZ=Europe/Zurich` wirkt also auch im Container, obwohl das Image keine Dateien für Zeitzonen mitbringt.
- **Der Healthcheck.** `easyplan healthcheck` fragt die laufende Anwendung und beendet sich mit 0 bei einer gesunden und mit 1 bei einer kranken Antwort. Der Datenbank-Ping hat zwei Sekunden Zeit, der Client drei, Docker fünf.

## Dokumentation

Die Doku liegt unter [`docs/`](docs/) und wird mit [MkDocs](https://www.mkdocs.org/) und dem Material-Theme gebaut.

```bash
python3 -m venv .venv-docs
. .venv-docs/bin/activate
pip install -r requirements-docs.txt
mkdocs serve     # Vorschau auf http://127.0.0.1:8000
mkdocs build --strict
```

Inhalt:

- [Überblick und Status](docs/index.md)
- [Architektur](docs/architecture.md)
- [Oberfläche](docs/features/oberflaeche.md)
- [Dashboard](docs/features/dashboard.md)
- [Einträge](docs/features/eintraege.md)
- [Timeline](docs/features/timeline.md)
- [Konten](docs/features/konten.md)
- [Formate](docs/features/formate.md)
- [Server und Betrieb](docs/features/server.md)
- [Sicherheit](docs/features/sicherheit.md)
- [Berechnung](docs/features/berechnung.md)
- [Wiederholungen](docs/features/wiederholungen.md)
- [Datenspeicher](docs/features/datenspeicher.md)
- [Betrieb mit Docker](docs/betrieb.md)
- [Backup und Restore](docs/backup.md)
- [CI und Dependabot](docs/ci.md)
- [Architekturentscheidungen](docs/adr/index.md)
- [ADR-0001: SQLite ohne CGO](docs/adr/0001-sqlite-ohne-cgo.md)
- [ADR-0002: Rappen und Rundung](docs/adr/0002-rappen-und-rundung.md)
- [ADR-0003: Kalendertage in UTC](docs/adr/0003-kalendertage-utc.md)
- [ADR-0004: Nächste Zahlung pro Konto](docs/adr/0004-naechste-zahlung-pro-konto.md)
- [ADR-0005: Origin-Prüfung statt CSRF-Token](docs/adr/0005-origin-pruefung-statt-csrf-token.md)
- [ADR-0006: Zeitraum und Monatsgruppen der Timeline](docs/adr/0006-timeline-zeitraum-und-monatsgruppen.md)
- [ADR-0007: Backup als POST](docs/adr/0007-backup-als-post.md)
- [ADR-0008: Bind-Default und Image-Override](docs/adr/0008-bind-default-und-image-override.md)
- [ADR-0009: Update-Policy für Go](docs/adr/0009-dependabot-update-policy-go.md)
- [Beitragen](docs/contributing.md)
- [Changelog](docs/changelog.md)

## Lizenz

MIT, siehe [LICENSE](LICENSE). Copyright (c) 2026 HeiligerG.
