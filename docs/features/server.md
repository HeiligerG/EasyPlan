# Server und Betrieb

EasyPlan ist ein Binary. Es startet einen HTTP-Server, öffnet eine SQLite-Datei und sonst nichts. Weitere Dienste, ein Build-Schritt fürs Frontend und eine Einstellungsseite gibt es nicht.

## Starten

```bash
EASYPLAN_DATA=./data go run ./cmd/easyplan
```

Der Server hört danach auf `EASYPLAN_ADDR` und legt die Datenbank im Datenordner an. Beim Start schreibt er eine Zeile ins Log:

```text
time=2026-10-31T19:00:00.000+01:00 level=INFO msg="easyplan started" addr=:8080 data=./data database=data/easyplan.db tz=Europe/Zurich
```

Die Zeitstempel und die Pfade hängen von der Umgebung ab. Der Log geht nach `stderr`, das Logformat ist `log/slog` im Textformat.

## Die Konfiguration

Die Konfiguration kommt aus der Umgebung. Es gibt keine Einstellungsseite und keine Konfigurationsdatei.

| Variable | Standard | Bedeutung |
|---|---|---|
| `EASYPLAN_ADDR` | `:8080` | Die Adresse, auf der der Server hört |
| `EASYPLAN_DATA` | `/data` | Der Ordner mit der SQLite-Datei |
| `TZ` | `Europe/Zurich` | Die Zeitzone der Anwendung |

Eine leere Variable zählt wie nicht gesetzt, dann greift der Standardwert.

!!! warning "`:8080` hört auf allen Schnittstellen"
    Der Standardwert `:8080` bindet an alle Schnittstellen des Rechners, nicht nur an `127.0.0.1`. EasyPlan hat keine eigene Authentifizierung, der Zugriff soll ein Reverse Proxy regeln. Wer die App direkt auf einem Rechner startet, sollte sie deshalb nur auf dem Loopback hören lassen:

    ```bash
    EASYPLAN_ADDR=127.0.0.1:8080 EASYPLAN_DATA=./data go run ./cmd/easyplan
    ```

    `Konzept.md` §9 verlangt dasselbe: Der Dienst lauscht nur auf `127.0.0.1`, Zugriff und TLS regelt ein Reverse Proxy. Das Beispiel für einen Container mit Compose folgt mit Phase 3.

## Der Datenordner

Der Ordner aus `EASYPLAN_DATA` wird beim Start angelegt, wenn er noch nicht existiert (Modus 0750). Darin liegt die SQLite-Datei `easyplan.db`.

Die Migrationen laufen beim Start automatisch. Ist das Schema der Datei neuer als das Binary unterstützt, bricht der Start mit einer klaren Meldung ab. Die Einzelheiten stehen unter [Datenspeicher](datenspeicher.md).

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

## Das Beenden

Ein `SIGINT` (Ctrl+C) oder ein `SIGTERM` beenden den Server sauber:

1. Der Server nimmt keine neuen Anfragen mehr an.
2. Die Anfragen, die noch laufen, bekommen zehn Sekunden Zeit.
3. Danach wird die Datenbank geschlossen.

Im Log steht am Ende `easyplan stops`.

## Was noch fehlt

- Der Unterbefehl `healthcheck` ist ein Platzhalter. `easyplan healthcheck` bricht mit `subcommand "healthcheck" is not implemented yet` ab.
- Die Adresse `/health` gibt es noch nicht. Sie antwortet wie jede unbekannte Adresse mit der Seite für 404.
- Container, Compose-Datei, Backup und Wiederherstellung folgen mit Phase 3.

## Weiterführend

- [Sicherheit](sicherheit.md): die Regeln, die der Server bei jedem Schreibzugriff anwendet
- [Architektur](../architecture.md): die Bausteine und ihre Abhängigkeiten
