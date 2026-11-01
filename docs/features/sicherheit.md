# Sicherheit

EasyPlan hat keine eigene Authentifizierung. Der Zugriff regelt ein Reverse Proxy, so wie es `Konzept.md` §9 vorsieht. Was die Anwendung selbst schützt, sind die schreibenden Zugriffe: sie kommen nur an, wenn sie vom selben Ursprung stammen.

## Die Prüfung der Herkunft

Bei jedem schreibenden Request (`POST`, `PUT`, `PATCH`, `DELETE`) prüft der Server den Header `Origin`:

- Der Header muss gesetzt sein. Ein Request ohne `Origin` wird abgelehnt.
- Der Host im `Origin` muss gleich sein wie der `Host` der Anfrage, mit Port und ohne Beachtung der Gross- und Kleinschreibung.
- Das Schema wird ignoriert. Hinter einem Proxy, der TLS beendet, passt das.
- `Origin: null` (der Ursprung eines sandboxed Frames) gehört zu niemandem und wird abgelehnt.

Passt die Herkunft nicht, antwortet der Server mit dem Status 403 und der Seite „Zugriff verweigert“. Die Details stehen im Log, nicht auf der Seite.

!!! warning "Ein Reverse Proxy muss den `Host`-Header erhalten"
    Der Vergleich ist immer gegen den `Host`-Header der Anfrage. `X-Forwarded-Host` wird nicht ausgewertet.

    - Caddy und Traefik setzen den `Host`-Header von Haus aus weiter, es ist nichts zu tun.
    - nginx braucht dafür eine Zeile in der Proxy-Konfiguration:

    ```nginx
    proxy_set_header Host $host;
    ```

    Ohne erhaltenen `Host` sind alle Schreibzugriffe 403, und die App sieht nur noch lesen.

Diese Regel ist als [ADR-0005: Origin-Prüfung statt CSRF-Token](../adr/0005-origin-pruefung-statt-csrf-token.md) festgehalten.

## Die Header

Jede Antwort trägt drei Header:

| Header | Wert |
|---|---|
| `Content-Security-Policy` | `default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; form-action 'self'; base-uri 'self'; frame-ancestors 'none'; object-src 'none'` |
| `X-Content-Type-Options` | `nosniff` |
| `Referrer-Policy` | `same-origin` |

Die Content-Security-Policy erlaubt nur Ressourcen der Anwendung selbst. Es gibt kein Inline-Script und kein Inline-Style: Das Stylesheet liegt als Datei unter `/static/style.css`, das Script von HTMX als Datei unter `/static/htmx.min.js`. Inline-Styles des Browsers werden deshalb von den Templates nicht benutzt.

## Die Ausgabe

Alle Seiten entstehen mit `html/template`, nicht mit `text/template`. Der Escaping des Pakets schützt die Werte, die aus der Datenbank oder aus einer Eingabe stammen, vor HTML und Attributen.

Die Zahlungen und Beträge sind Text und keine Markup-Elemente, darum wird der Apostroph des Tausendertrenners escaped. Der Browser zeigt ihn wieder als `'` an.

## Die Regeln für die Schreibzugriffe

- Keine `GET`-Route verändert Daten. Alle Aktionen, die etwas ändern, sind `POST`-Requests auf eigenen Adressen.
- Nach einem erfolgreichen Schreibzugriff antwortet der Server mit dem Status 303 und einer Adresse. Das Muster Post/Redirect/Get verhindert, dass ein Neuladen die Aktion wiederholt.
- Der Body eines Formulars ist auf 64 KiB begrenzt. Ein grösserer Body wird mit dem Status 413 abgelehnt.
- Die Eingaben werden serverseitig geprüft, bevor etwas gespeichert wird. Die Prüfung der Formulare ist eine reine Funktion und steht unter [Einträge](eintraege.md) und [Konten](konten.md).

## Das Backup als Datenabfluss

Ein Backup enthält die ganze Datenbank: alle Konten, alle Einträge, alle Beträge und alle Notizen. Es ist damit der einzige Ort, an dem die Anwendung Daten ausliefert, und er ist besonders zu schützen.

Die Adresse `POST /backup` ist darum die einzige schreibende Adresse, die Daten ausliefert:

- Die Adresse antwortet nur auf `POST`. Ein `GET /backup` kennt keine Route und liefert die Seite für 404. Ein `GET` muss nach RFC 9110 §9.2.1 sicher sein, ein Backup ist es nicht: Es kopiert die ganze Datenbank und erzeugt Last.
- Weil `POST` eine schreibende Methode ist, greift die Prüfung der Herkunft auch hier. Ein Request ohne `Origin` oder mit fremdem `Origin` wird mit dem Status 403 abgelehnt. Das ist der zusätzliche Schutz für die schreibenden Methoden: Eine fremde Seite kann das Backup nicht anstossen.
- Der Download trägt `Cache-Control: no-store`, damit keine Kopie in einem Zwischencache liegen bleibt.
- Der Knopf steht im Seitenfuss und heisst „Backup herunterladen“. Die Datei bekommt einen Namen mit der Zeit des Backups.

Damit kann kein Link, kein Prefetch und kein Suchroboter ein Backup auslösen. Es braucht die Oberfläche selbst und eine passende Herkunft.

Dass eine fremde Seite die Antwort nicht lesen kann, regelt die Same-Origin-Policy des Browsers, unabhängig von der Methode. Die Prüfung der Herkunft verhindert zusätzlich, dass das Backup überhaupt anstossen wird.

Die Regel ist als [ADR-0007: Backup als POST](../adr/0007-backup-als-post.md) festgehalten. Der Ablauf steht unter [Backup](../backup.md).

## Die Fehlerseiten

Eine Fehlerseite verrät nichts über den Grund des Fehlers. Die Details gehen in das Log des Servers, die Seite bekommt einen allgemeinen Text wie „Die Seite konnte nicht geladen werden.“

## Unbekannte Adressen und falsche Methoden

Jede Adresse, die keine Route kennt, führt zur Seite für 404. Das gilt auch, wenn die Adresse bekannt ist, aber mit der falschen Methode aufgerufen wird: `PUT /entries` antwortet mit 404 und nicht mit 405.

Der Grund ist der Catch-all am Ende der Route-Liste. Er fängt jede Adresse ab, die keine Route bedient, und liefert dieselbe Seite. Der Besucher sieht damit nicht, welche Adressen und Methoden es gibt.

Die Prüfung der Herkunft läuft vor der Verteilung auf die Routen. Ein `POST /health` ohne passenden `Origin` wird deshalb mit 403 abgelehnt, mit passendem `Origin` mit 404, weil es die Adresse nur für `GET` gibt.

## Die statischen Dateien

CSS und HTMX liegen als eingebettete Dateien im Binary, es gibt keine Dateien neben dem Programm und keinen Build-Schritt. Die Dateien bekommen einen `ETag` aus dem SHA-256 des Inhalts und einen `Cache-Control`-Header mit einer Stunde. Der Browser lädt eine Datei nach einem Update des Binaries neu.

## Weiterführend

- [Server und Betrieb](server.md): Start, Konfiguration und der Hinweis zur Listen-Adresse
- [Backup](../backup.md): der Download der Datenbank und die Wiederherstellung
- [ADR-0005: Origin-Prüfung statt CSRF-Token](../adr/0005-origin-pruefung-statt-csrf-token.md)
- [ADR-0007: Backup als POST](../adr/0007-backup-als-post.md)
- [Formate](formate.md): die Grenzen der Eingaben
