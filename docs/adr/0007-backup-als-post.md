# ADR-0007: Backup als POST mit temporärer Datei im Datenordner

Status: accepted
Datum: 2026-10-31
Überarbeitet: 2026-11-01, die Begründung der Entscheidung wurde nach dem Review von Phase 3 korrigiert

## Kontext

`Konzept.md` §6 nennt eine fünfte Seite „Backup“ mit dem Button „Datenbank herunterladen“, §8 beschreibt den Ablauf: Der Button ruft `VACUUM INTO` auf eine temporäre Datei auf und liefert sie als Download.

Ein Backup kopiert die ganze Datenbank: Die Kopie enthält alle Konten, alle Einträge, alle Beträge und alle Notizen. Es erzeugt damit Last auf dem Server und liefert Daten aus.

Ein `GET` muss sicher sein (RFC 9110 §9.2.1): Der Server darf dabei keinen Zustand verändern. Eine `GET`-Adresse für das Backup wäre nicht sicher. Ein Link in einem Forum, ein Prefetch des Browsers, ein Suchroboter oder eine Link-Vorschau genügt, um die Kopie anzustossen, ohne dass die Besucherin es will.

Die Prüfung der Herkunft aus [ADR-0005](0005-origin-pruefung-statt-csrf-token.md) greift nur bei schreibenden Methoden (`POST`, `PUT`, `PATCH`, `DELETE`). Sie ist damit der zusätzliche Schutz für die schreibenden Methoden, nicht der Grund für die Wahl von `POST`.

Der Container läuft mit einem unbeschreibbaren Root-Dateisystem (`read_only: true`). Eine temporäre Datei unter `/tmp` kann dort nicht angelegt werden. Beschreibbar ist nur `/data`.

Offen war auch die Frage, ob das Backup eine eigene Seite bekommt oder ein Knopf auf einer bestehenden Seite ist.

## Entscheidung

- Das Backup antwortet nur auf `POST /backup`. Ein `GET /backup` kennt keine Route und liefert die Seite für 404. Grund ist die Sicherheit von `GET` nach RFC 9110 §9.2.1.
- Der Knopf heisst „Backup herunterladen“ und steht im Seitenfuss jeder Seite. Es gibt keine eigene Backup-Seite: Das Backup ist eine Aktion und keine Seite, die Inhalt anzeigt.
- Der Knopf erscheint nur, wenn der Server einen Ordner für die temporäre Kopie kennt. Ohne diesen Ordner antwortet die Adresse mit dem Status 503.
- Die Kopie entsteht mit `VACUUM INTO` (`store.VacuumInto`) in einem temporären Ordner im Datenordner (`Options.BackupDir`, im Betrieb `/data`).
- Der temporäre Ordner hat die Rechte 0700, die Kopie die Rechte 0600. Beides gehört dem Prozess allein.
- Der temporäre Ordner wird in jedem normalen Weg wieder gelöscht: nach dem Download, nach einem Fehler und nach einem Besucher, der die Verbindung abbricht. Das Löschen hängt an einem `defer`.
- Ein harter Abbruch des Programms kann das Löschen verhindern. Beim nächsten Start entfernt die Anwendung alle Verzeichnisse mit dem Präfix `easyplan-backup-` im Datenordner. Es werden nur Verzeichnisse entfernt, ein symbolischer Link wird nicht verfolgt, und ein Fehler beim Entfernen wird als Warnung geloggt und verhindert den Start nicht.
- Die Backup-Route setzt für ihre Antwort eine eigene Schreibfrist von zehn Minuten. Die allgemeine Schreibfrist des Servers von 60 Sekunden beginnt vor dem Kopieren der Datenbank und würde eine grosse Datenbank abschneiden. Die eigene Frist wirkt bei HTTP/1.1, im Betrieb spricht der Reverse Proxy HTTP/1.1 mit der Anwendung.

## Konsequenzen

- Das Backup ist nur aus der Oberfläche heraus erreichbar. Weil `POST` eine schreibende Methode ist, greift die Prüfung der Herkunft auch hier: Eine fremde Seite kann das Backup nicht anstossen.
- Dass eine fremde Seite die Antwort nicht lesen kann, regelt die Same-Origin-Policy des Browsers, unabhängig von der Methode. Die Prüfung der Herkunft schützt zusätzlich, sie ist nicht deren Ersatz.
- Werkzeuge ohne `Origin`-Header, etwa ein `curl` ohne `-H`, werden abgelehnt. Für eine Prüfung aus der Kommandozeile muss der Header mitgeschickt werden.
- Im Datenordner liegt für einen Moment eine Kopie der ganzen Datenbank. Der Ordner ist nur für den Prozess sichtbar und die Datei wird in jedem normalen Weg wieder entfernt.
- Nach einem harten Abbruch kann ein Verzeichnis `easyplan-backup-*` im Datenordner liegenbleiben. Es wird beim nächsten Start entfernt und ist damit kein bleibender Betriebsartefakt.
- Der Download braucht keinen zweiten beschreibbaren Ort und funktioniert darum auch im Container mit unbeschreibbarem Root-Dateisystem.
- Das Restore läuft von Hand: Container stoppen, WAL- und SHM-Datei des alten Standes entfernen, Datei mit `install` ersetzen, Container starten.
- `Konzept.md` §6 nennt noch eine eigene Seite 5 mit dem Text „Datenbank herunterladen“. Verbindlich sind der Code und diese Doku, die Abweichung ist hier festgehalten.
