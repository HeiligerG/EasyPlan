# ADR-0001: Eine SQLite-Datei ohne CGO (modernc.org/sqlite)

Status: accepted
Datum: 2026-10-30

## Kontext

EasyPlan soll als ein Binary laufen, das eine SQLite-Datei liest und schreibt. Weitere Dienste, ein ORM und ein Build-Schritt fürs Frontend sind ausgeschlossen.

Die Go-Standardbibliothek bringt keinen SQLite-Treiber mit. Der verbreitete Treiber `github.com/mattn/go-sqlite3` braucht CGO und damit einen C-Compiler. Das erzwingt beim Linux-Build eine C-Werkzeugkette, verhindert ein schlankes Container-Image und macht den Build abhängig von der Plattform.

## Entscheidung

EasyPlan nutzt `modernc.org/sqlite`, eine reine Go-Implementierung von SQLite, ohne CGO. Der Treiber wird in `internal/store` eingebunden.

- Eine Datenbank pro Instanz, als eine Datei im Datenordner. Der Ordner kommt aus `EASYPLAN_DATA`.
- Der Store öffnet die Datei über `store.Open(path)`, für Tests gibt es `store.OpenMemory()`.
- Die Pragmas stehen im Data Source Name: `foreign_keys(1)`, `journal_mode(WAL)` und `busy_timeout(5000)`. Der Treiber wendet sie auf jede Verbindung an, ein Pragma auf einer einzelnen Verbindung wäre beim nächsten Verbindungswechsel verloren.
- Die Migrationen liegen als SQL-Dateien im Binary (`embed`) und laufen beim Start. Die Schemaversion steht in `PRAGMA user_version`.
- Alle offenen Migrationen eines Starts laufen in einer Transaktion, und die Version wird innerhalb dieser Transaktion gelesen. Ein zweiter Prozess wartet deshalb auf den ersten und kann dieselbe Migration nicht doppelt anwenden.
- Eine Datenbank, deren Schema neuer ist als die Migrationen des Binaries, wird abgelehnt. Das Binary weiss nicht, wie das neuere Schema aussieht.

## Konsequenzen

- Das Binary baut auf jedem Rechner mit Go, ohne C-Compiler und ohne Plattformabhängigkeiten. Ein Image ohne Systembibliotheken ist möglich.
- Direkte Abhängigkeit ist nur `modernc.org/sqlite` (Version 1.61.0). Die weiteren Module in `go.mod` sind indirekte Abhängigkeiten dieses Treibers.
- Der Treiber ist langsamer als ein C-Treiber. Für eine einzelne Nutzerin mit einer Handvoll Einträge ist das ohne Bedeutung.
- Die In-Memory-Datenbank der Tests kann kein WAL, der Store fragt es dort nicht an.
- Die Inhalte der Datenbank bleiben für die Nutzerin lesbar, weil es eine Standarddatei ohne proprietären Zusatz ist.
