# ADR-0009: Update-Policy für Go (Patch automatisch, Minor und Major von Hand)

Status: accepted
Datum: 2026-10-31

## Kontext

Dependabot meldet neue Versionen von Modulen, Basis-Images, Aktionen und Werkzeugen. Für die Go-Version gibt es keine Abhängigkeit, die Dependabot von sich aus aktualisieren könnte: Im Ökosystem `gomod` sind nur die Modulpfade Abhängigkeiten, die `go`-Direktive in `go.mod` ist es nicht.

Die Go-Standardbibliothek steckt statisch im ausgelieferten Binary. Patch-Releases bringen fast immer Korrekturen für Sicherheitslücken in `net/http` und `crypto/tls` mit. Sie ankommen nur, wenn jemand sie einholt.

Ein Minor- oder Major-Wechsel der Go-Version ist anders: Er verlangt eine neue `go`-Direktive in `go.mod` und ein neues Tag im `Dockerfile`. Zwei Dateien, eine bewusste Entscheidung, ein Review.

## Entscheidung

- Patch-Updates des Builder-Images (1.27.3 zu 1.27.4) sind erlaubt und laufen als eigener Pull Request. Die Gruppe `docker-images` nimmt das Go-Image deshalb ausdrücklich aus (`exclude-patterns`), damit der Patch nicht an einem anderen Image-Update hängt.
- Minor- und Major-Updates der Go-Version sind gesperrt. Die Sperre liegt in der `ignore`-Regel des Ökosystems `docker` und nennt nur `version-update:semver-minor` und `version-update:semver-major`.
- Minor- und Major-Wechsel erfolgen von Hand, in einem einzigen Commit, der `go.mod` und das `Dockerfile` zusammen ändert.
- Die `go`-Direktive in `go.mod` ist eine Mindestversion. Das ist der Grund, warum ein Patch des Builder-Images gültig bleibt.

## Konsequenzen

- Die Korrekturen der Standardbibliothek kommen ohne Warten auf Menschen an, mit dem üblichen Review durch den CI-Workflow.
- Ein Patch des Builder-Images ändert nur das Tag im `Dockerfile`. `go.mod` bleibt, weil die Mindestversion weiterhin erfüllt ist.
- Ein Minor-Wechsel braucht zwei Dateien und damit eine Handarbeit. Das ist gewollt: Er kann das Verhalten des Programms beeinflussen.
- Steigt die `go`-Direktive in `go.mod` über die Go-Version des Builder-Images, braucht der Build eine neuere Toolchain. Der Job `docker` in der CI zeigt das, weil er das Image mit dem Builder baut.
- Die Abhängigkeiten der Module (`modernc.org/sqlite` und die indirekten) bleiben im Ökosystem `gomod` und werden weiterhin automatisch als Minor- oder Patch-Update gemeldet.
