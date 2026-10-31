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

## Die Fehlerseiten

Eine Fehlerseite verrät nichts über den Grund des Fehlers. Die Details gehen in das Log des Servers, die Seite bekommt einen allgemeinen Text wie „Die Seite konnte nicht geladen werden.“

## Unbekannte Adressen und falsche Methoden

Jede Adresse, die keine Route kennt, führt zur Seite für 404. Das gilt auch, wenn die Adresse bekannt ist, aber mit der falschen Methode aufgerufen wird: `PUT /entries` antwortet mit 404 und nicht mit 405.

Der Grund ist der Catch-all am Ende der Route-Liste. Er fängt jede Adresse ab, die keine Route bedient, und liefert dieselbe Seite. Der Besucher sieht damit nicht, welche Adressen und Methoden es gibt.

## Die statischen Dateien

CSS und HTMX liegen als eingebettete Dateien im Binary, es gibt keine Dateien neben dem Programm und keinen Build-Schritt. Die Dateien bekommen einen `ETag` aus dem SHA-256 des Inhalts und einen `Cache-Control`-Header mit einer Stunde. Der Browser lädt eine Datei nach einem Update des Binaries neu.

## Weiterführend

- [Server und Betrieb](server.md): Start, Konfiguration und der Hinweis zur Listen-Adresse
- [ADR-0005: Origin-Prüfung statt CSRF-Token](../adr/0005-origin-pruefung-statt-csrf-token.md)
- [Formate](formate.md): die Grenzen der Eingaben
