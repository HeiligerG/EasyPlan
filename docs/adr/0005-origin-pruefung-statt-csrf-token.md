# ADR-0005: Origin-Prüfung statt CSRF-Token

Status: accepted
Datum: 2026-10-31

## Kontext

`Konzept.md` §9 verlangt einen CSRF-Schutz ohne Token: Bei jedem schreibenden Request (POST, PUT, DELETE) prüft der Server, dass der `Origin`-Header zum `Host` passt. Die Eingaben werden serverseitig validiert.

EasyPlan hat keine eigene Authentifizierung, den Zugriff regelt ein Reverse Proxy. Fremde Seiten im Browser des Nutzers können aber trotzdem Formulare an die Anwendung schicken. Ein solcher Request landet beim Reverse Proxy wie jeder andere und wäre ohne weitere Prüfung ein Schreibzugriff.

Es gab drei Wege:

- **Ein Token im Formular.** Das Token muss mit einem vom Server gehaltenen Wert übereinstimmen. Das verlangt eine Sitzung: entweder ein Cookie mit einer Sitzungskennung oder ein Cookie mit einem signierten Token. EasyPlan hält heute keinen Zustand ausser der SQLite-Datei.
- **SameSite-Cookies.** Wirkt nur, wenn die Anwendung überhaupt Cookies setzt, und nur in Browsern, die SameSite richtig umsetzen.
- **Die Prüfung der Herkunft.** Der Browser sendet bei jedem schreibenden Request den `Origin`-Header. Der Server vergleicht ihn mit dem Host, an den der Request ging.

Die Browser senden den `Origin` bei Formularen und bei `fetch`-Aufrufen immer, auch wenn die Anfrage vom selben Ursprung kommt. Der Header ist also zuverlässig vorhanden, wenn die Anfrage aus der Oberfläche stammt.

## Entscheidung

Bei `POST`, `PUT`, `PATCH` und `DELETE` prüft die Web-Schicht die Herkunft der Anfrage:

- Der Header `Origin` muss gesetzt sein. Ein Request ohne `Origin` wird mit dem Status 403 abgelehnt.
- Der Host aus dem `Origin` muss mit dem `Host` der Anfrage übereinstimmen, mit Port und ohne Beachtung der Gross- und Kleinschreibung.
- Das Schema des `Origin` wird ignoriert, damit die Prüfung auch hinter einem Proxy funktioniert, der TLS beendet.
- `Origin: null` hat keinen Host und gehört zu niemandem. Der Wert wird abgelehnt.
- Verglichen wird immer gegen den `Host`-Header. `X-Forwarded-Host` wird nicht ausgewertet.

Die Ablehnung antwortet mit dem Status 403 und einer Fehlerseite ohne Details. Die Gründe stehen im Log.

## Konsequenzen

- Die Anwendung bleibt zustandslos: keine Sitzung, keine Cookies, kein Token.
- Ein Reverse Proxy muss den `Host`-Header erhalten. Caddy und Traefik tun das von Haus aus, nginx braucht `proxy_set_header Host $host;`. Ohne erhaltenen `Host` sind alle Schreibzugriffe 403. Das steht in der [Doku zur Sicherheit](../features/sicherheit.md).
- Werkzeuge ohne `Origin`-Header, etwa ein `curl` ohne `-H`, werden abgelehnt. Das ist beabsichtigt: Die Schnittstelle ist für den Browser gedacht. Für Prüfungen aus der Kommandozeile muss der Header mitgeschickt werden.
- Die Prüfung schützt nur gegen Anfragen aus einem fremden Ursprung. Sie ersetzt keine Authentifizierung, die regelt weiterhin der Reverse Proxy.
- Der Aufwand liegt in der Middleware (`internal/web/origin.go`), nicht in den einzelnen Formularen. Jeder schreibende Request läuft durch sie, egal auf welcher Adresse.
