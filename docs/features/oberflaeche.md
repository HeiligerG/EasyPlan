# Oberfläche

Die Oberfläche besteht aus vier Seiten, die über die Navigation im Kopf verbunden sind. Alle Seiten sind Server-Seiten: Der Browser bekommt HTML, das der Server gerendert hat.

## Die Seiten

| Seite | Adresse | Inhalt |
|---|---|---|
| Dashboard | `/` | Bedarf, Reserve und nächste Zahlung pro Konto |
| Einträge | `/entries` | Liste der Einträge, Anlegen, Bearbeiten, Löschen |
| Timeline | `/timeline` | Die Zahlungen der nächsten zwölf Monate |
| Konten | `/accounts` | Anlegen, Umbenennen, Sortieren, Deaktivieren |

Die Seiten im Detail stehen unter [Dashboard](dashboard.md), [Einträge](eintraege.md), [Timeline](timeline.md) und [Konten](konten.md).

Jede Seite hat denselben Aufbau: ein Kopf mit der Marke und der Navigation, der Hauptbereich mit der Seite und ein Fuss. Der Eintrag der aktuellen Seite in der Navigation ist markiert.

## Der Knopf im Fuss

Der Fuss jeder Seite trägt den Knopf „Backup herunterladen“. Er lädt die SQLite-Datei als Datei herunter.

```html
<form class="site-footer__backup" method="post" action="/backup">
<button type="submit" class="secondary">Backup herunterladen</button>
</form>
```

Der Knopf ist ein normales Formular ohne JavaScript, wie alle anderen Formulare der Anwendung. Er steht im Fuss und nicht auf einer eigenen Seite, weil ein Backup eine Aktion ist und keine Seite mit Inhalt.

Der Knopf erscheint nur, wenn der Server den Backup einrichten kann. Ohne einen Ordner für die temporäre Kopie gibt es den Knopf nicht, und die Adresse antwortet mit dem Status 503.

Die Adresse antwortet nur auf `POST`: Ein `GET` muss nach RFC 9110 §9.2.1 sicher sein, ein Backup ist es nicht, weil es die ganze Datenbank kopiert und Last erzeugt. Die Einzelheiten stehen unter [Backup](../backup.md) und [Sicherheit](sicherheit.md).

## HTMX

HTMX 2.0.11 liegt als einzelne Datei im Binary und wird ausgeliefert, es gibt keine CDN-Abhängigkeit zur Laufzeit:

```text
Quelle: https://unpkg.com/htmx.org@2.0.11/dist/htmx.min.js
Lizenz: Zero-Clause BSD (0BSD)
```

Die Datei wird mit `defer` geladen, die Seite ist also auch ohne JavaScript vollständig nutzbar. Aktuell nutzt die Oberfläche kein einziges `hx-*`-Attribut: Alle Formulare und Links sind gewöhnliche HTML-Elemente. HTMX ist damit vorbereitet, aber noch nicht eingesetzt.

Weil die Content-Security-Policy kein Inline-Script erlaubt, trägt der Kopf ein Meta-Element mit der Konfiguration:

```html
<meta name="htmx-config" content='{"allowEval":false,"includeIndicatorStyles":false}'>
```

- `allowEval: false` verbietet das Auswerten von Attributen mit `hx-on` und von `eval`.
- `includeIndicatorStyles: false` unterdrückt das Nachladen der HTMX-Styles, weil die Vorgabe von `style-src 'self'` kein Inline-Style erlaubt.

## Das Stylesheet

Die Oberfläche hat genau ein CSS-File, `internal/web/static/style.css`. Es ist eingebettet, es gibt kein Frontend-Build und kein Werkzeug ausser Go.

- Mobile first: Das Layout beginnt mit einer Spalte und wird erst mit einer Media Query breiter (ab 40 rem).
- Kein Dark Mode, keine Charts, kein Framework, keine Web-Fonts.
- Die Farben stehen als CSS-Variablen im Kopf der Datei.
- Die Tabellen des Dashboards und der Einträge bekommen einen Mindestbreite, damit sie auf kleinen Schirmen waagrecht scrollen statt die Spalten zu quetschen.
- Jeder Knopf und jeder Knopf als Link ist mindestens 44 Pixel hoch (`min-height: 44px`). Die Fläche zum Tippen ist damit auf jedem Bildschirm gross genug. Die Regel zählt den Rand und den Innenabstand mit, der Knopf sieht darum gleich aus und wird nur etwas höher.

## Ohne JavaScript

Jede Seite funktioniert ohne JavaScript:

- Die Navigation sind Links.
- Die Formulare sind normale `POST`-Formulare mit einem Redirect danach.
- Die Auswahl der Reserve auf dem Dashboard und der Kontofilter der Timeline sind Links beziehungsweise ein `GET`-Formular.
- Das Löschen eines Eintrags braucht eine Bestätigungsseite und damit keine Erweiterung des Browsers.

## Die Sprache

Die Oberfläche ist auf Deutsch (Schweiz). Datumsangaben stehen als `TT.MM.JJJJ`, Beträge in ganzen Franken mit dem Apostroph als Tausendertrenner. Die Einzelheiten stehen unter [Formate](formate.md).

## Weiterführend

- [Sicherheit](sicherheit.md): die Header und die Prüfung der Herkunft
- [Backup](../backup.md): der Download der Datenbank aus dem Seitenfuss
- [Architektur](../architecture.md): der Aufbau des Pakets `internal/web`
