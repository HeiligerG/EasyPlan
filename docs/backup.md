# Backup

Ein Backup von EasyPlan ist die SQLite-Datei selbst. Es gibt keinen Export, kein Format und keine Konfiguration: Der Button im Seitenfuss liefert eine Kopie der Datenbank, und das Restore läuft von Hand.

Die Anleitung für den Betrieb mit Container steht unter [Betrieb](betrieb.md).

## Der Button

Im Seitenfuss jeder Seite steht der Button „Backup herunterladen“. Er ist ein normales Formular ohne JavaScript:

```html
<form class="site-footer__backup" method="post" action="/backup">
<button type="submit" class="secondary">Backup herunterladen</button>
</form>
```

Der Button erscheint nur, wenn der Server einen Ordner für die temporäre Kopie kennt. Ohne diesen Ordner gibt es den Button nicht, und ein Aufruf der Adresse antwortet mit dem Status 503.

## Warum POST und nicht GET

Ein `GET` muss sicher sein (RFC 9110 §9.2.1): Der Server darf dabei keinen Zustand verändern. Ein Backup verändert zwar keinen Zustand der Anwendung, es erzeugt aber Last und liefert die ganze Datenbank aus: alle Konten, alle Einträge, alle Beträge, alle Notizen. Als `GET` ausgelöst könnte es ein Link im Forum, ein Prefetch des Browsers, ein Suchroboter oder eine Link-Vorschau auslösen, ohne dass die Besucherin es will.

Darum antwortet die Adresse nur auf `POST`:

- Ein `GET /backup` gibt es nicht. Die Adresse wird mit der Seite für 404 beantwortet.
- `POST` ist eine schreibende Methode. Für sie gilt die Prüfung der Herkunft: Der Request muss den Header `Origin` mitbringen, und der Host im `Origin` muss zum `Host` der Anfrage passen. Sonst antwortet der Server mit dem Status 403.

Ein Link, ein Prefetch des Browsers oder ein Suchroboter löst damit kein Backup aus. Nur ein Formular aus der Oberfläche heraus kann es.

Die Prüfung der Herkunft ist der zusätzliche Schutz für die schreibenden Methoden: Eine fremde Seite kann das Backup nicht anstossen. Dass eine fremde Seite die Antwort nicht lesen kann, regelt die Same-Origin-Policy des Browsers, und zwar unabhängig von der Methode.

Diese Regel ist als [ADR-0007: Backup als POST](adr/0007-backup-als-post.md) festgehalten.

## Was der Server tut

Der Ablauf im Code (`internal/web/backup.go`):

1. Der Server legt einen temporären Ordner im Datenordner an (`os.MkdirTemp`). Der Ordner hat die Rechte 0700, gehört also nur dem Prozess.
2. In diesen Ordner schreibt er die Kopie der Datenbank (`store.VacuumInto`, siehe [Datenspeicher](features/datenspeicher.md)).
3. Die Kopie bekommt die Rechte 0600. Niemand sonst auf dem Rechner kann sie lesen.
4. Die Datei wird als Download geschickt.
5. Der temporäre Ordner wird wieder gelöscht. Das geschieht in jedem normalen Weg: nach dem Download, nach einem Fehler und auch dann, wenn der Besucher die Verbindung abbricht.

Schritt 5 hängt an einem `defer`. Der Ordner bleibt damit nur so lange liegen, wie die Anfrage dauert, und im Datenordner bleibt nichts zurück.

### Was nach einem Absturz liegen bleibt

Wird das Programm hart beendet, etwa mit `kill -9` oder durch einen Stromausfall, kann es den temporären Ordner nicht mehr löschen. Im Datenordner bleibt dann ein Verzeichnis mit dem Präfix `easyplan-backup-` liegen.

Beim nächsten Start räumt die Anwendung den Datenordner auf, noch bevor der Server annimmt. Sie entfernt jedes Verzeichnis, dessen Name mit `easyplan-backup-` beginnt:

- Es werden nur Verzeichnisse entfernt. Dateien mit diesem Namen bleiben liegen.
- Ein symbolischer Link wird nicht verfolgt und nicht entfernt.
- Schlägt das Entfernen fehl, schreibt die Anwendung eine Warnung ins Log. Der Start läuft weiter.

Ein solches Verzeichnis gehört damit zum Betrieb und ist kein Zeichen eines Fehlers.

### Die Schreibfrist der Antwort

Der Server hat eine allgemeine Schreibfrist von 60 Sekunden (`WriteTimeout`). Sie beginnt, bevor die Kopie der Datenbank gemacht ist, und würde eine grosse Datenbank abschneiden: Die Antwort fängt erst an, wenn die Kopie fertig ist.

Darum setzt die Backup-Route für ihre Antwort eine eigene Frist von zehn Minuten. Sie gilt für die Kopie und für den Download zusammen.

Die Frist wirkt bei HTTP/1.1. Im Betrieb spricht der Reverse Proxy HTTP/1.1 mit der Anwendung, darum greift sie dort. Kann der Server die Frist nicht setzen, läuft die Antwort ohne sie weiter, das ist kein Fehler.

### Warum der Ordner im Datenordner liegt

Das Root-Dateisystem des Containers ist unbeschreibbar (`read_only: true`). Eine temporäre Datei unter `/tmp` kann der Container dort nicht anlegen. Der Datenordner `/data` ist die einzige Stelle, an der geschrieben werden darf, und deshalb schreibt der Server die Kopie dorthin.

Der Download liest die Kopie von der Platte und schreibt sie in die Antwort. Er braucht darum keinen zweiten beschreibbaren Ort.

## Die Antwort

| Angabe | Wert |
|---|---|
| Status | 200 |
| `Content-Type` | `application/vnd.sqlite3` |
| `Content-Disposition` | `attachment; filename="easyplan-YYYYMMDD-HHMMSS.db"` |
| `Cache-Control` | `no-store` |
| `Content-Length` | Die Grösse der Kopie |

Der Dateiname trägt die Zeit des Backups in der Zeitzone der Anwendung, im Format `JJJJMMTT-HHMMSS`. Zwei Backups am selben Tag haben darum verschiedene Namen.

`Cache-Control: no-store` verhindert, dass ein Zwischencache die Kopie aufhebt. Sie enthält alle Daten und soll nach dem Download nirgends sonst liegen.

Die Kopie entsteht mit `VACUUM INTO`. Sie ist eine vollständige, in sich geschlossene Datenbank: Man kann sie öffnen, ohne die Dateien daneben zu brauchen.

## Wenn etwas schiefgeht

| Fall | Status | Was zu sehen ist |
|---|---|---|
| Der Server hat keinen Ordner für die Kopie | 503 | Die Seite „Backup nicht verfügbar“ mit dem Hinweis „Das Backup ist nicht eingerichtet.“ Der Button fehlt dann. |
| Die Kopie kann nicht geschrieben werden | 500 | Die allgemeine Fehlerseite. Der Grund steht im Log. |
| Die Herkunft passt nicht | 403 | Die Seite „Zugriff verweigert“. |
| `GET /backup` | 404 | Die Seite für eine unbekannte Adresse. |
| Der Besucher bricht ab | 200 | Die Antwort ist unvollständig. Der Abbruch steht im Log, die temporäre Datei wird trotzdem gelöscht. |

Die Seite sagt in keinem Fall, warum etwas schiefgegangen ist. Die Einzelheiten stehen im Log des Servers.

## Das Restore von Hand

Das Restore läuft von Hand, es gibt keinen Upload in der Oberfläche.

!!! warning "Vor dem Replace den Container stoppen"
    Der Container muss stehen, während die Datei ersetzt wird. Läuft er weiter, schreibt er in die alte Datei, während sie ersetzt wird, und die Datenbank kann beschädigt sein.

Die Schritte:

```bash
# 1. Container stoppen
docker compose stop

# 2. Die WAL- und SHM-Datei des alten Standes entfernen.
sudo rm -f ./data/easyplan.db-wal ./data/easyplan.db-shm

# 3. Die Backup-Datei an den Platz der Datenbank legen. Der Name ist der
#    aus dem Download, hier als Beispiel:
sudo install -o 65532 -g 65532 -m 600 \
  ~/Downloads/easyplan-20261031-214500.db ./data/easyplan.db

# 4. Container starten
docker compose up -d
```

Der Befehl `install` ersetzt die Datei und setzt Besitzer, Gruppe und Rechte im selben Schritt. Die Datenbank liegt damit nie mit fremden Rechten da, auch nicht für einen Moment. Zwei Aufrufe (`cp` und danach `chown`) wären der umständlichere Weg mit demselben Ergebnis.

Danach prüfen:

```bash
docker compose ps
curl -fsS http://127.0.0.1:8080/health
```

### Warum die WAL- und SHM-Datei weg müssen

Die Datenbank läuft im WAL-Modus. Neben `easyplan.db` liegen deshalb `easyplan.db-wal` und `easyplan.db-shm`. Sie gehören zum alten Stand der Datenbank.

Bleiben sie liegen, nimmt SQLite sie beim Öffnen mit und spielt die alten Änderungen auf die neue Datei. Das Ergebnis ist eine Mischung aus altem und neuem Stand, und im schlimmsten Fall eine beschädigte Datenbank. Die beiden Dateien gehören deshalb zum alten Stand und werden beim Restore entfernt.

Die Backup-Datei selbst braucht keine Begleitdateien: `VACUUM INTO` schreibt eine vollständige Datenbank in eine einzige Datei.

### Nach dem Restore

Der Container startet mit der wiederhergestellten Datei und prüft, ob das Schema zum Binary passt. Ist die Datei aus einem neueren Stand, bricht der Start ab, statt die Daten anzufassen. In dem Fall ist die Binärdatei zu alt und muss zuerst aktualisiert werden, siehe [Betrieb](betrieb.md).

## Automatische Backups

Die Oberfläche ist für den Betrieb gedacht, nicht für ein Backup-Programm. Für regelmässige Kopien gibt es zwei Wege.

### Der Download über die Kommandozeile

Die Adresse des Backups lässt sich auch von aussen aufrufen. Der Header `Origin` muss mitgeschickt werden, sonst antwortet der Server mit 403:

```bash
curl -fsS -X POST \
  -H "Origin: http://127.0.0.1:8080" \
  http://127.0.0.1:8080/backup \
  -o "easyplan-$(date +%Y%m%d-%H%M%S).db"
```

Der Host der Adresse und der Host im `Origin` müssen übereinstimmen, sonst verweigert die Anwendung die Anfrage. Ein Cron-Job mit diesem Aufruf genügt, und er braucht keinen Zugriff auf den Datenordner.

### Die Kopie mit dem Werkzeug `sqlite3`

Wer die Datei direkt kopieren will, nimmt das Werkzeug `sqlite3` auf dem Rechner:

```bash
sqlite3 ./data/easyplan.db ".backup '/pfad/zum/ziel/easyplan-$(date +%Y%m%d-%H%M%S).db'"
```

Der Befehl `.backup` nimmt einen konsistenten Stand der Datenbank, ohne den Container anzuhalten. Er braucht das Werkzeug `sqlite3` auf dem Rechner, nicht im Container, und Schreibzugriff auf den Datenordner: Eine Datenbank im WAL-Modus braucht die gemeinsame Speicherdatei `easyplan.db-shm`, und die gehört dem Nutzer 65532.

## Weiterführend

- [Betrieb](betrieb.md): der Container, der Datenordner und das Update
- [Sicherheit](features/sicherheit.md): die Prüfung der Herkunft, die das Backup schützt
- [Datenspeicher](features/datenspeicher.md): der WAL-Modus und die Begleitdateien
- [ADR-0007: Backup als POST](adr/0007-backup-als-post.md)
