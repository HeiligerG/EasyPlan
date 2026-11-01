# Betrieb

EasyPlan ist ein Binary mit einer SQLite-Datei. Im Betrieb kommt ein Container dazu: Er startet das Binary, prüft selbst, ob er gesund ist, und schreibt in genau einen Ordner. Alles andere bleibt unbeschreibbar.

Diese Seite beschreibt den Betrieb mit Docker. Die Datensicherung hat ihre eigene Seite unter [Backup](backup.md), die automatischen Prüfungen und Updates die Seite [CI und Dependabot](ci.md).

## Das Container-Image

Das Image wird aus dem `Dockerfile` im Repository gebaut. Es besteht aus zwei Stufen:

| Stufe | Image | Aufgabe |
|---|---|---|
| Bau | `golang:1.27.3-bookworm` | Kompiliert das Binary ohne CGO |
| Laufzeit | `gcr.io/distroless/static-debian12:nonroot` | Führt das Binary aus, ohne Shell und ohne Root |

Beide Basis-Images sind an einen Digest gepinnt: Die `FROM`-Zeile nennt Tag und Digest, das Image bleibt also auch dann dasselbe, wenn jemand das Tag verschiebt. Die Pin pflegt Dependabot im Ökosystem `docker`: Ein Update bringt Tag und Digest gemeinsam in den Pull Request. Die Regel für die Go-Version steht unter [CI und Dependabot](ci.md).

Das Binary wird statisch und ohne Debug-Symbole gebaut:

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/easyplan ./cmd/easyplan
```

Die Laufzeit-Stufe hat keine Shell, kein Paketverwaltung und kein `curl`. Der Healthcheck ruft deshalb das eigene Unterkommando des Binaries auf, statt ein Werkzeug im Image zu brauchen.

### Was die Laufzeit-Stufe einstellt

| Angabe | Wert | Bedeutung |
|---|---|---|
| `USER` | `65532:65532` | Der Nutzer `nonroot` des distroless-Images |
| `ENV EASYPLAN_ADDR` | `:8080` | Der Server hört im Container auf allen Schnittstellen |
| `ENV EASYPLAN_DATA` | `/data` | Der Ordner der SQLite-Datei |
| `ENV TZ` | `Europe/Zurich` | Die Zeitzone der Anwendung |
| `VOLUME` | `/data` | Der Ordner, in den geschrieben wird |
| `EXPOSE` | `8080` | Der Port, den der Server benutzt |
| `ENTRYPOINT` | `/easyplan` | Das Binary |

Der Container läuft als `nonroot` und nicht als `root`. Es gibt keine Dateien neben dem Programm: Das Stylesheet, das Script von HTMX und die Templates liegen eingebettet im Binary.

!!! info "Warum die Zeitzone im Container funktioniert"
    Das distroless-Image bringt keine Dateien für Zeitzonen mit. Das Binary bindet die Zeitzonendatenbank selbst ein (`time/tzdata`), darum wirkt `TZ=Europe/Zurich` auch ohne `/usr/share/zoneinfo`. Eine unbekannte Zone bricht den Start ab, mit derselben Meldung wie ausserhalb des Containers.

## Der Datenordner

Der Container schreibt in genau einen Ordner: `/data`. Dort liegen die SQLite-Datei und ihre Begleitdateien.

```text
/data/
├── easyplan.db        die Datenbank
├── easyplan.db-wal    das Write-Ahead-Log
└── easyplan.db-shm    die gemeinsame Speicherdatei
```

Die beiden Begleitdateien gehören zum WAL-Modus von SQLite und liegen neben der Datenbank. Sie sind für den Betrieb normal und kein Zeichen eines Fehlers.

Beim Start entfernt die Anwendung ausserdem die Verzeichnisse, die ein abgebrochenes Backup liegen gelassen hat. Sie heissen `easyplan-backup-*` und enthielten eine Kopie der ganzen Datenbank. Das Aufräumen betrifft nur Verzeichnisse, ein symbolischer Link wird nicht verfolgt. Schlägt es fehl, schreibt die Anwendung eine Warnung ins Log und startet trotzdem. Die Einzelheiten stehen unter [Backup](backup.md).

### Die Rechte des Ordners

Der Nutzer `65532` muss den Ordner besitzen, sonst kann der Container die Datenbank nicht anlegen. Vor dem ersten Start:

```bash
sudo chown 65532:65532 ./data
```

Die Rechte des Ordners selbst bleiben, der Inhalt gehört dem Container. Weitere Rechte sind nicht nötig, und `chmod 777` ist die falsche Antwort.

### Kein Netzwerkdateisystem

!!! warning "WAL funktioniert nicht auf Netzwerkdateisystemen"
    Der WAL-Modus braucht eine gemeinsame Speicherdatei (`easyplan.db-shm`), die über eine Dateischnittstelle abgebildet wird. Auf NFS und auf anderen Netzwerkdateisystemen ist das nicht zuverlässig. Ein Datenordner auf einem Netzwerklaufwerk ist darum nicht unterstützt.

    Wer die Daten auf einem Netzlaufwerk braucht, legt sie lokal ab und kopiert sie von dort.

## Die Compose-Datei

`compose.yaml` im Repository beschreibt den Dienst:

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
      TZ: Europe/Zurich
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
```

Die Zeilen, die den Betrieb tragen:

| Angabe | Wirkung |
|---|---|
| `read_only: true` | Das Root-Dateisystem des Containers ist unbeschreibbar. Geschrieben wird nur `/data`. |
| `127.0.0.1:8080:8080` | Der Port ist nur auf dem Loopback des Hosts erreichbar, nicht im Netz. |
| `./data:/data` | Die Daten liegen im Repository-Ordner und überstehen ein Neu bauen des Containers. |
| `restart: unless-stopped` | Der Container startet nach einem Neustart des Hosts wieder. |
| `security_opt` / `cap_drop` | `no-new-privileges:true` und `cap_drop: [ALL]` nehmen dem Container alle Rechte, die er nicht braucht. |
| `image: easyplan:local` | Der Name des lokal gebauten Images. Compose baut es aus dem `Dockerfile` und versieht es mit diesem Namen. |
| `stop_grace_period: 15s` | Die Zeit, die der Container beim Stoppen bekommt. Sie ist länger als die zehn Sekunden, die das Binary für das Beenden braucht, damit der Server seine Anfragen noch beenden kann. |
| `logging` | Die Rotation des Container-Logs: `json-file` mit `max-size: 10m` und `max-file: 3`. Das Log bleibt damit auf etwa 30 MB begrenzt. |

`EASYPLAN_ADDR` und `EASYPLAN_DATA` setzt das Image selbst, die Compose-Datei wiederholt sie nicht.

!!! warning "`docker compose pull` funktioniert hier nicht"
    Das Image `easyplan:local` wird lokal gebaut, es gibt es in keiner Registry. `docker compose pull` sucht es dort und findet nichts. Gebaut wird mit `docker compose build` oder mit `docker compose up -d --build`.

### Die Portbindung im LAN

`docker run -p 8080:8080` ohne den Teil `127.0.0.1:` bindet den Port auf alle Schnittstellen des Hosts. Die Anwendung ist dann im Netz erreichbar, ohne Anmeldung und ohne TLS, und mit ihr auch der Download des Backups mit der ganzen Datenbank.

Die Form `127.0.0.1:8080:8080` ist darum Pflicht. Sie bindet den Port nur auf dem Loopback des Hosts, von aussen braucht es einen Reverse Proxy, siehe [Sicherheit](features/sicherheit.md).

## Die Installation

### Die Schritte

```bash
git clone https://github.com/HeiligerG/EasyPlan.git
cd EasyPlan

# Der Datenordner gehört dem Nutzer 65532 (nonroot) des Containers.
mkdir -p ./data
sudo chown 65532:65532 ./data

docker compose up -d
```

`docker compose up -d` baut das Image, falls es noch nicht existiert, startet den Dienst im Hintergrund und kehrt sofort zurück.

### Die Prüfung

Zuerst die Antwort der Anwendung:

```bash
curl -fsS http://127.0.0.1:8080/health
```

```text
ok
```

Die Antwort ist `ok` mit dem Status 200, wenn die Datenbank erreichbar ist. Ist sie es nicht, antwortet der Dienst mit `unhealthy` und dem Status 503.

Dann der Zustand des Containers:

```bash
docker compose ps
```

```text
NAME       IMAGE            STATUS                PORTS
easyplan-1 easyplan:local   Up 2 minutes (healthy) 127.0.0.1:8080->8080/tcp
```

In der Spalte `STATUS` steht `healthy`, sobald der Healthcheck des Containers erfolgreich war. Nach einem Neustart des Containers kann es bis zu zehn Sekunden dauern, bis der erste Check läuft.

Wer sehen will, dass der Container ausserhalb von `/data` nichts schreibt, vergleicht das Dateisystem des Containers mit dem Image:

```bash
docker diff "$(docker compose ps -q easyplan)"
```

Die Ausgabe ist leer. Der Container beschreibt nur `/data`, und `/data` ist ein Bind-Mount, also kein Teil des Dateisystems des Containers.

## Der Healthcheck

### Der Unterbefehl

Das Binary bringt einen eigenen Unterbefehl mit:

```bash
easyplan healthcheck
```

Der Aufruf fragt die laufende Anwendung, ob sie gesund ist, und beendet sich danach:

| Exit-Code | Bedeutung |
|---|---|
| `0` | Die Anwendung antwortet mit dem Status 200. |
| `1` | Die Adresse ist ungültig, die Antwort kam nicht, war unvollständig oder trug einen anderen Status. |

Das Docker-Image benutzt genau diesen Aufruf, in der Exec-Form, weil das distroless-Image keine Shell hat:

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/easyplan", "healthcheck"]
```

Die Bedeutungen der vier Werte:

| Wert | Bedeutung |
|---|---|
| `--interval=30s` | Zwischen zwei Checks liegen dreissig Sekunden. |
| `--timeout=5s` | Ein Check, der nach fünf Sekunden keine Antwort hat, gilt als fehlgeschlagen. |
| `--start-period=10s` | In den ersten zehn Sekunden nach dem Start zählt ein Fehlschlag nicht. |
| `--retries=3` | Drei Fehlschläge in Folge machen den Container `unhealthy`. |

### Die Adresse des Checks

Der Check benutzt die Adresse aus `EASYPLAN_ADDR`. Der Host dieser Adresse wird übernommen, der Port wird immer mitgeprüft:

| `EASYPLAN_ADDR` | Adresse des Checks |
|---|---|
| `:8080` | `http://127.0.0.1:8080/health` |
| `127.0.0.1:8080` | `http://127.0.0.1:8080/health` |
| `0.0.0.0:8080` | `http://127.0.0.1:8080/health` |
| `[::]:8080` | `http://127.0.0.1:8080/health` |
| `plan.example.com:8080` | `http://plan.example.com:8080/health` |

Eine Adresse ohne Host oder mit dem unspezifischen Host `0.0.0.0` oder `::` hört auf allen Schnittstellen. Der Check fragt in dem Fall den Loopback ab, weil nur dort bekannt ist, auf welchem Port gespannt wird.

`plan.example.com` steht in der Tabelle für einen eigenen Host.

Der Port 0 wird abgelehnt: Er bedeutet, dass der Server einen freien Port wählt, der Check wüsste nicht, wo er suchen soll.

### Die Timeouts

Drei Timeouts stecken in der Prüfung, sie liegen alle knapp untereinander:

| Timeout | Wert | Ort | Wirkung |
|---|---|---|---|
| Datenbank | 2 s | `internal/web/health.go` | So lange darf die Datenbank für den Ping brauchen. |
| Client | 3 s | `cmd/easyplan/main.go` | So lange wartet der Client auf die ganze Antwort. |
| Container | 5 s | `Dockerfile` | So lange wartet die Container-Runtime auf den Check. |

Die Kette ist so gebaut, dass der Client schneller abbricht als die Container-Runtime. Ein hängender Datenbankzugriff wird also als Fehlschlag gemeldet und nicht als gesund festgehalten.

### Das Log

Der Healthcheck schreibt nichts, wenn die Anwendung gesund ist. Bei einer kaputten Datenbank schreibt die Anwendung eine Zeile mit dem Level `Warn` ins Log:

```text
time=2026-10-31T22:14:07.000+01:00 level=WARN msg="the health check failed" error="store: cannot reach the database: ..."
```

Die Zeile steht auf `Warn` und nicht auf `Error`: Die Container-Runtime fragt alle dreissig Sekunden, ein Fehler käme sonst im Sekundentakt als Fehler im Log.

Bei einer dauerhaft kaputten Datenbank entsteht damit alle dreissig Sekunden eine Zeile. Das ist beabsichtigt: Der Zustand soll im Log stehen, auch wenn niemand auf die Oberfläche schaut.

## Das Update

```bash
git pull
docker compose build
docker compose up -d
```

Die drei Schritte bedeuten:

1. `git pull` holt den neuen Stand des Codes.
2. `docker compose build` baut das Image neu. Die Abhängigkeiten werden nur neu geladen, wenn sich `go.mod` oder `go.sum` geändert haben.
3. `docker compose up -d` ersetzt den Container durch einen mit dem neuen Image.

Ein `docker compose pull` ist hier nicht sinnvoll: Das Image wird lokal gebaut und gibt es in keiner Registry. Wer beide Schritte in einem will, nimmt `docker compose up -d --build`.

Die Basis-Images werden nicht von Hand aktualisiert. Dependabot bringt neue Digests in einem Pull Request, der mit dem `Dockerfile` kommt, siehe [CI und Dependabot](ci.md).

Die Daten bleiben im Bind-Mount `./data`. Das Update ersetzt nur das Programm, die Datenbank wird nicht angefasst.

Der Start des neuen Containers führt die offenen Migrationen aus. Ist die Datenbank neuer als das Binary, bricht der Start mit einer klaren Meldung ab und nimmt die Daten nicht an. Das ist die gewollte Sicherung gegen einen Verlust von Daten.

## Was der Container nicht kann

- **Keine Authentifizierung.** EasyPlan hat keine eigene Anmeldung. Der Zugriff gehört hinter einen Reverse Proxy, siehe [Sicherheit](features/sicherheit.md).
- **Kein Speicher ausserhalb von `/data`.** Das Root-Dateisystem ist unbeschreibbar, auch nicht für ein Log.
- **Kein Netzwerkdateisystem.** Siehe die Warnung zum WAL-Modus weiter oben.
- **Keine mehreren Instanzen auf derselben Datei.** Zwei gleichzeitige Erststarts auf einer brandneuen Datei können sporadisch `SQLITE_BUSY` melden, siehe [Datenspeicher](features/datenspeicher.md).

## Weiterführend

- [Backup](backup.md): die Datensicherung und die Wiederherstellung
- [CI und Dependabot](ci.md): die automatischen Prüfungen und die Aktualisierungen
- [Sicherheit](features/sicherheit.md): die Regeln, die der Server bei jedem Schreibzugriff anwendet
- [Server und Betrieb](features/server.md): Start, Konfiguration und Beenden ohne Container
