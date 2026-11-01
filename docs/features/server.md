# Server und Betrieb

EasyPlan ist ein Binary. Es startet einen HTTP-Server, öffnet eine SQLite-Datei und sonst nichts. Weitere Dienste, ein Build-Schritt fürs Frontend und eine Einstellungsseite gibt es nicht.

## Starten

```bash
EASYPLAN_DATA=./data go run ./cmd/easyplan
```

Der Server hört danach auf `EASYPLAN_ADDR` und legt die Datenbank im Datenordner an. Beim Start schreibt er eine Zeile ins Log:

```text
time=2026-10-31T19:00:00.000+01:00 level=INFO msg="easyplan started" addr=127.0.0.1:8080 data=./data database=data/easyplan.db tz=Europe/Zurich
```

Die Zeitstempel und die Pfade hängen von der Umgebung ab. Der Log geht nach `stderr`, das Logformat ist `log/slog` im Textformat.

## Die Konfiguration

Die Konfiguration kommt aus der Umgebung. Es gibt keine Einstellungsseite und keine Konfigurationsdatei.

| Variable | Standard ausserhalb von Docker | Wert im Image | Bedeutung |
|---|---|---|---|
| `EASYPLAN_ADDR` | `127.0.0.1:8080` | `:8080` | Die Adresse, auf der der Server hört |
| `EASYPLAN_DATA` | `/data` | `/data` | Der Ordner mit der SQLite-Datei `easyplan.db` |
| `TZ` | `Europe/Zurich` | `Europe/Zurich` | Die Zeitzone der Anwendung |

Eine leere Variable zählt wie nicht gesetzt, dann greift der Standardwert.

!!! note "Der Standardwert ist der Loopback"
    Ausserhalb von Docker hört der Server auf `127.0.0.1:8080`. EasyPlan hat keine eigene Authentifizierung, der Zugriff soll ein Reverse Proxy regeln. Der Standardwert hält die App deshalb von aussen zu.

    Wer sie breiter hören lassen will, setzt die Variable ausdrücklich:

    ```bash
    EASYPLAN_ADDR=:8080 EASYPLAN_DATA=./data go run ./cmd/easyplan
    ```

    Im Container-Image ist `EASYPLAN_ADDR` auf `:8080` gestellt, weil der Server dort auf allen Schnittstellen des Containers hören muss, damit der gemappte Port ankommt. Dort entscheidet der Host, was erreichbar ist. Die Regel ist als [ADR-0008: Bind-Default und Image-Override](../adr/0008-bind-default-und-image-override.md) festgehalten.

`EASYPLAN_DATA` ist auch ausserhalb von Docker `/data`. Für die Entwicklung setzt man deshalb `EASYPLAN_DATA=./data`, sonst braucht das Anlegen des Ordners Root-Rechte.

## Der Datenordner

Der Ordner aus `EASYPLAN_DATA` wird beim Start angelegt, wenn er noch nicht existiert (Modus 0750). Darin liegt die SQLite-Datei `easyplan.db`, im WAL-Modus daneben die Dateien `easyplan.db-wal` und `easyplan.db-shm`.

Die Migrationen laufen beim Start automatisch. Ist das Schema der Datei neuer als das Binary unterstützt, bricht der Start mit einer klaren Meldung ab. Die Einzelheiten stehen unter [Datenspeicher](datenspeicher.md).

## Die Prüfung der Gesundheit

### Die Adresse `/health`

`GET /health` beantwortet eine einfache Frage: Ist die Anwendung gesund?

| Fall | Status | Antwort |
|---|---|---|
| Die Datenbank ist erreichbar | 200 | `ok` |
| Die Datenbank ist nicht erreichbar | 503 | `unhealthy` |

Die Antwort ist reiner Text (`text/plain; charset=utf-8`), keine Seite der Oberfläche. Die Einzelheiten eines Fehlers stehen im Log, nicht in der Antwort.

Die Prüfung braucht die Datenbank (`store.Ping`) und schreibt nichts. Ein Healthcheck verändert also keinen Zustand.

Die Aussage der Prüfung ist begrenzt: Der Treiber beantwortet sie mit einem `select 1`, also einer Konstante. Die Prüfung liest keine Daten der Datenbank, sie prüft nicht die Datei auf der Platte und sie macht keine Schreibprobe. Eine gelöschte Datei bei offener Verbindung wird darum nicht erkannt.

Schlägt die Prüfung fehl, antwortet der Server mit 503 und schreibt eine Zeile mit dem Level `Warn` ins Log, nicht mit `Error`: Die Container-Runtime fragt alle dreissig Sekunden, ein Fehler käme sonst im Sekundentakt als Fehler im Log.

### Der Unterbefehl `healthcheck`

Das Binary bringt einen eigenen Unterbefehl mit, mit dem eine Container-Runtime oder ein Skript die laufende Anwendung fragen kann:

```bash
easyplan healthcheck
```

| Exit-Code | Bedeutung |
|---|---|
| `0` | Die Anwendung antwortet mit dem Status 200. |
| `1` | Die Adresse war ungültig, die Antwort kam nicht, war unvollständig oder trug einen anderen Status. |

Der Unterbefehl nimmt die Adresse aus `EASYPLAN_ADDR` und fragt `http://<Host>:<Port>/health`. Der Host kommt aus der Adresse, ein leerer Host, `0.0.0.0` und `::` bedeuten „hört auf allen Schnittstellen“ und werden zu `127.0.0.1`. Der Port 0 wird abgelehnt: Er bedeutet, dass der Server einen freien Port wählt.

Eine IPv6-Adresse mit Zonen-ID, etwa `[fe80::1%eth0]:8080`, wird für den Check maskiert: Das Prozentzeichen der Zonen-ID wird zu `%25`, weil die sonst ungültige URL vom HTTP-Client abgelehnt würde.

### Die Timeouts der Prüfung

| Timeout | Wert | Ort | Wirkung |
|---|---|---|---|
| Datenbank | 2 s | `internal/web/health.go` | So lange darf die Datenbank für den Ping brauchen. |
| Client | 3 s | `cmd/easyplan/main.go` | So lange wartet der Client auf die ganze Antwort. |
| Container | 5 s | `Dockerfile` | So lange wartet die Container-Runtime auf den Check. |

Die Kette ist so gebaut, dass der Client schneller abbricht als die Container-Runtime. Der Betrieb mit Docker steht unter [Betrieb](../betrieb.md).

## Die Zeitzone

Die Zone aus `TZ` wird beim Start mit `time.LoadLocation` geladen:

- Eine unbekannte Zone bricht den Start ab: `easyplan: cannot load the time zone "Mars/Phobos": unknown time zone Mars/Phobos`.
- Die Zone wird einmal beim Start gesetzt und gilt für den ganzen Prozess. Die Web-Schicht bekommt sie zusätzlich über `Options.Location`, damit die Handler sie nicht über die globale Variable `time.Local` beziehen müssen.
- Der heutige Tag wird in der Zone der Anwendung bestimmt: er beginnt um Mitternacht in dieser Zone, nicht um Mitternacht UTC. Die gespeicherten Daten und die berechneten Termine sind dagegen Kalendertage ohne Uhrzeit, siehe [ADR-0003: Kalendertage in UTC](../adr/0003-kalendertage-utc.md).

Die Zeitzonendatenbank ist im Binary eingebettet (`time/tzdata`). Darum funktioniert `TZ=Europe/Zurich` auch in einem Container, der keine System-Datenbank für Zeitzonen mitbringt.

## Die Timeouts des Servers

| Timeout | Wert | Wirkung |
|---|---|---|
| `ReadHeaderTimeout` | 10 s | Wie lange der Server auf die Kopfzeilen wartet |
| `ReadTimeout` | 30 s | Wie lange der Server den Body liest |
| `WriteTimeout` | 60 s | Wie lange der Server die Antwort schreibt |
| `IdleTimeout` | 120 s | Wie lange eine offene Verbindung zwischen zwei Anfragen leben darf |

Die Backup-Route setzt für ihre Antwort eine eigene Schreibfrist von zehn Minuten. Die allgemeine Frist von 60 Sekunden beginnt, bevor die Kopie der Datenbank gemacht ist, und würde eine grosse Datenbank abschneiden. Die eigene Frist wirkt bei HTTP/1.1; im Betrieb spricht der Reverse Proxy HTTP/1.1 mit der Anwendung.

## Das Beenden

Ein `SIGINT` (Ctrl+C) oder ein `SIGTERM` beenden den Server sauber:

1. Der Server nimmt keine neuen Anfragen mehr an.
2. Die Anfragen, die noch laufen, bekommen zehn Sekunden Zeit.
3. Danach wird die Datenbank geschlossen.

Im Log steht am Ende `easyplan stops`.

## Weiterführend

- [Betrieb](../betrieb.md): Container, Compose-Datei, Healthcheck und Update
- [Backup](../backup.md): die Datensicherung und die Wiederherstellung
- [Sicherheit](sicherheit.md): die Regeln, die der Server bei jedem Schreibzugriff anwendet
- [Architektur](../architecture.md): die Bausteine und ihre Abhängigkeiten
