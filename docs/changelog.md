# Changelog

Alle Änderungen an EasyPlan stehen in diesem Dokument. Das Format folgt [Keep a Changelog](https://keepachangelog.com/de/1.1.0/), die Versionierung folgt [SemVer](https://semver.org/lang/de/).

Diese Datei ist die einzige Quelle für den Changelog. Eine zweite Kopie im Repository-Root gibt es nicht.

## [Unreleased]

### Added

- Bedarf pro Konto und gesamt: Monat, Woche und Tag, abgeleitet aus den Jahreskosten der Einträge.
- Reserve pro Konto und gesamt für die nächsten N Tage. Jede Fälligkeit im Zeitraum zählt, auch wenn derselbe Eintrag mehrfach vorkommt.
- Nächste Zahlung pro Konto: der früheste Termin aller aktiven Einträge, der heutige Tag eingeschlossen.
- Wiederholungen in den Einheiten einmalig, wöchentlich, monatlich und jährlich, mit einem Intervall bis 10'000 Einheiten.
- Monatsende-Regel mit Schaltjahr: ein monatlicher Eintrag vom 31.01. läuft am 28.02. weiter und kommt am 31.03. zurück.
- Ende eines Eintrags über ein optionales Enddatum, das am Termin selbst noch gilt.
- SQLite-Datenspeicher mit eingebetteten Migrationen, Fremdschlüsseln und WAL-Modus.
- Standardkonten Lifestyle, Sparen und Steuern entstehen mit der ersten Migration. Ein gelöschtes Standardkonto kommt nicht zurück.
- Der Start bricht mit einer klaren Meldung ab, wenn die Datenbank ein neueres Schema hat als das Binary unterstützt.
- Zwei gleichzeitige Starts auf einer bereits angelegten Datenbankdatei legen die Tabellen nicht doppelt an. Beim ersten Start auf einer brandneuen Datei kann sporadisch `SQLITE_BUSY` auftreten.
- Die Eingaben werden geprüft: Beträge bis 10^12 Rappen (10 Milliarden Franken), Datumswerte in den Jahren 1 bis 9999. Das Enddatum liegt nie vor dem Startdatum.
- Kontoverwaltung: anlegen, umbenennen, sortieren, deaktivieren und löschen, solange das Konto keine Einträge hat.
- Dokumentation mit MkDocs: Überblick, Architektur, Berechnung, Wiederholungen, Datenspeicher, ADRs und Changelog.

### Changed

- Keine.

### Fixed

- Keine.

### Removed

- Keine.

### Security

- Keine.

### Noch nicht vorhanden

Die Benutzeroberfläche mit Dashboard, Einträgen, Konten und Timeline folgt mit Phase 2, der Betrieb mit Container, Backup und Installation mit Phase 3. Bis dahin gibt es keine lauffähige App, siehe [Status](index.md).
