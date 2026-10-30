# Datenspeicher

EasyPlan speichert alles in einer einzigen SQLite-Datei. Das Paket `internal/store` hält die Migrationen und alle Abfragen, das ORM gibt es nicht, die SQL-Abfragen stehen direkt im Code.

Die Datei wird mit `store.Open(path)` geöffnet, der Ordner muss bereits existieren. Für Tests gibt es `store.OpenMemory()` mit einer Datenbank im Arbeitsspeicher.

## Schema

Die Migration `internal/store/migrations/0001_init.sql` legt zwei Tabellen an und füllt die Standardkonten ein:

```sql
CREATE TABLE accounts (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active  INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE entries (
    id           INTEGER PRIMARY KEY,
    account_id   INTEGER NOT NULL REFERENCES accounts(id),
    name         TEXT    NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    start_date   TEXT    NOT NULL,
    unit         TEXT    NOT NULL CHECK (unit IN ('once', 'week', 'month', 'year')),
    interval     INTEGER NOT NULL DEFAULT 1 CHECK (interval > 0),
    ends_on      TEXT    NULL,
    notes        TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_entries_account_id ON entries(account_id);

INSERT INTO accounts (name, sort_order, is_active) VALUES
    ('Lifestyle', 1, 1),
    ('Sparen', 2, 1),
    ('Steuern', 3, 1);
```

- Beträge stehen als positive Rappen in `amount_minor`. Das Minuszeichen entsteht erst bei der Anzeige.
- Datumswerte stehen als Text im Format `YYYY-MM-DD` ohne Uhrzeit.
- `unit` kennt nur die vier Einheiten, `interval` ist immer grösser als 0.
- `ends_on` ist `NULL`, wenn der Eintrag kein Enddatum hat.
- Die Fremdschlüssel der Tabelle `entries` zeigen auf `accounts`.

## Migrationen

Die Migrationsdateien liegen als SQL-Dateien im Verzeichnis `internal/store/migrations` und werden mit `embed` in das Binary gepackt. Beim Start prüft der Store die Version des Schemas und führt nur die neueren Migrationen aus.

- Die Version steht im Kopf der Datenbankdatei (`PRAGMA user_version`). Eine zusätzliche Tabelle dafür gibt es nicht.
- Die Version einer Migration steht im Dateinamen: `0001_init.sql` ist die Version 1. Zwei Migrationen mit derselben Version werden abgelehnt, ihre Reihenfolge wäre nicht definiert: `duplicate migration version 1: 0001_init.sql and 0001_other.sql`.
- Alle offenen Migrationen eines Starts laufen in **einer** Transaktion. Schemaänderung und neue Version gelten entweder beide oder gar nicht.
- Die Version wird innerhalb dieser Transaktion gelesen. Die Transaktion nimmt das Schreibrecht sofort, ein zweiter Prozess wartet deshalb auf den ersten und sieht danach schon die neue Version. So kann dieselbe Migration nicht doppelt angewendet werden. Zwei gleichzeitige Starts auf einer bereits angelegten Datenbankdatei bestehen beide.
- Ist die Datenbank **neuer** als das Binary, bricht der Start mit einer klaren Meldung ab: `database schema version 99 is newer than this binary supports (1)`. Das Binary weiss nicht, wie das neuere Schema aussieht, und fasst die Daten deshalb nicht an.
- Eine bereits aktuelle Datenbank bleibt beim Öffnen unverändert, der Start ist also beliebig oft möglich.

!!! note "Zwei gleichzeitige Erststarts"
    Starten zwei Prozesse gleichzeitig auf einer brandneuen Datei, kann der zweite schon beim Verbindungsaufbau (Pragma `journal_mode(WAL)`, Anlegen der Datei) sporadisch `SQLITE_BUSY` melden, also noch vor jeder Migration. Auf einer existierenden Datei passiert das nicht. Im Betrieb mit einem Container ist das irrelevant, weil dort nur ein Prozess läuft.

## Standardkonten

Die Standardkonten entstehen mit der ersten Migration, nicht in einem eigenen Schritt danach:

| Name | Reihenfolge |
|---|---:|
| Lifestyle | 1 |
| Sparen | 2 |
| Steuern | 3 |

Damit gibt es sie genau einmal in jeder neuen Datenbank. Ein gelöschtes Standardkonto kommt nicht zurück, auch nicht beim nächsten Start. Umbenennen, deaktivieren und Sortieren bleiben erhalten.

## Verbindungen und Pragmas

Die Pragmas stehen im Data Source Name, weil der Treiber sie auf jede Verbindung anwendet. Ein Pragma, das nur auf einer Verbindung gesetzt wäre, wäre beim nächsten Verbindungswechsel weg.

```text
file:/pfad/easyplan.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_txlock=immediate
```

- `foreign_keys(1)`: Die Fremdschlüssel werden durchgesetzt.
- `journal_mode(WAL)`: Lesende Abfragen laufen parallel zu einem Schreibvorgang. Die In-Memory-Datenbank der Tests kann kein WAL und fragt es deshalb nicht an.
- `busy_timeout(5000)`: SQLite wartet fünf Sekunden auf einen gesperrten Schreibzugriff, statt sofort zu scheitern.
- `_txlock=immediate`: Eine Transaktion nimmt das Schreibrecht sofort, statt es später anzufordern.

Die Dateidatenbank bekommt vier Verbindungen, die In-Memory-Datenbank nur eine, weil jede Verbindung dort eine eigene leere Datenbank bekäme.

## Konten verwalten

Konten lassen sich anlegen, umbenennen, sortieren und deaktivieren. Die Einträge eines deaktivierten Kontos bleiben lesbar, fallen aber aus den Auswertungen heraus. Ein Konto mit Einträgen lässt sich nicht löschen, sondern nur deaktivieren.

```go
accounts, err := store.ListAccounts(ctx, false) // nur aktive Konten
err = store.SetAccountActive(ctx, id, false)    // deaktivieren
err = store.DeleteAccount(ctx, id)              // nur ohne Einträge
```

Die Namen der Konten sind case-sensitiv eindeutig: „Lifestyle“ und „lifestyle“ sind zwei verschiedene Konten.

## Einträge verwalten

Einträge werden vor dem Schreiben mit `plan.Entry.Validate` geprüft, der Store speichert also keinen Eintrag, den die Berechnung nicht verwenden könnte. Das Startdatum und das Enddatum laufen dabei immer über `plan.ParseDate` und `plan.FormatDate`, ein fehlendes Enddatum wird als `NULL` gespeichert. Die Grenzen der Eingaben stehen unter [Berechnung](berechnung.md) und [Wiederholungen](wiederholungen.md).

```go
id, err := store.CreateEntry(ctx, entry)
err = store.UpdateEntry(ctx, entry) // die Kennung im Eintrag entscheidet
err = store.DeleteEntry(ctx, id)
```

Die Abfragen liefern `store.Entry`. Der Typ bettet `plan.Entry` ein und ergänzt den Namen und den Zustand des Kontos (`AccountName`, `AccountActive`), damit die Auswertung ohne zweite Abfrage auskommt.

## Fehler

Der Store meldet Fehler über Sentinel-Werte, die mit `errors.Is` geprüft werden:

| Fehler | Bedeutung |
|---|---|
| `ErrNotFound` | Konto oder Eintrag existiert nicht |
| `ErrDuplicateName` | Der Kontoname ist bereits vergeben |
| `ErrInvalidName` | Der Kontoname ist leer |
| `ErrAccountInUse` | Das Konto hat Einträge und kann nicht gelöscht werden |
| `ErrUnknownAccount` | Der Eintrag verweist auf ein unbekanntes Konto |

Verletzungen der Datenbank-Constraints, etwa ein Betrag von 0 Franken oder eine unbekannte Einheit, meldet der Store als `invalid entry`.

## Backup

Ein Backup über `VACUUM INTO` und der Download-Button folgen mit Phase 3. Der Store bietet dafür heute noch keinen Zugriff auf die Datenbankverbindung.

## Weiterführend

- [ADR-0001: SQLite ohne CGO](../adr/0001-sqlite-ohne-cgo.md)
- [Architektur](../architecture.md): Bausteine und Datenfluss
