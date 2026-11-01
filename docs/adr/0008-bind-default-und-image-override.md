# ADR-0008: Bind-Default `127.0.0.1:8080` und Image-Override `:8080`

Status: accepted
Datum: 2026-10-31

## Kontext

`Konzept.md` §8 führt die Variable `EASYPLAN_ADDR` mit dem Standardwert `:8080` auf. §9 verlangt dagegen: „Der Dienst lauscht nur auf `127.0.0.1`. Zugriff und TLS regelt ein Reverse Proxy.“

EasyPlan hat keine eigene Authentifizierung. Ein Server, der auf allen Schnittstellen hört, ist deshalb sofort erreichbar, sobald jemand an den Rechner kommt.

Im Container ist es umgekehrt: Der Server muss auf allen Schnittstellen des Containers hören, sonst kommt der gemappte Port nie an. Die Beschränkung auf den Loopback muss dort der Host übernehmen, nicht das Programm.

Der Healthcheck braucht ausserdem eine Adresse, die er ansprechen kann, ohne den Port zu raten.

## Entscheidung

- Ausserhalb eines Containers hört der Server auf `127.0.0.1:8080`. Das ist der Standardwert von `EASYPLAN_ADDR` im Binary (`defaultAddr`).
- Das Image setzt `EASYPLAN_ADDR=:8080`. Im Container hört der Server damit auf allen Schnittstellen, aber nur der Host entscheidet, was erreichbar ist.
- Die Compose-Datei bindet nur `127.0.0.1:8080:8080` auf dem Host. Der Dienst ist damit auch im Container nur über den Loopback erreichbar.
- `EASYPLAN_ADDR` bleibt überschreibbar. Wer den Server breiter hören lassen will, setzt die Variable ausdrücklich.
- Der Healthcheck nimmt den Host aus `EASYPLAN_ADDR`. Ein leerer Host, `0.0.0.0` und `::` bedeuten „hört auf allen Schnittstellen“ und werden zu `127.0.0.1` für den Check. Der Port 0 wird abgelehnt, weil der Server dann einen freien Port wählt.

## Konsequenzen

- `go run ./cmd/easyplan` ist sicher per Default: Der Server ist nur über den Loopback erreichbar. Der Betrieb mit einem Reverse Proxy braucht keine zusätzliche Einstellung.
- Wer die App direkt auf einem Rechner startet und von aussen erreichbar machen will, setzt `EASYPLAN_ADDR` ausdrücklich und stellt den Zugriff selbst durch einen Proxy sicher.
- Im Container ist der Standardwert `:8080` von `Konzept.md` §8 erfüllt. Die Beschränkung auf `127.0.0.1` entsteht auf dem Host durch die Portbindung der Compose-Datei.
- Der Wert des Standards hängt von der Umgebung ab. Dokumentation und README nennen beide Werte und sagen, welcher wo gilt.
- Der Healthcheck meldet für jede gültige Bind-Adresse „gesund“. Er kann nicht wegen einer Adresse fälschlich „unhealthy“ melden.
