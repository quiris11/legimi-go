# legimi-go

Prosty, alternatywny program do pobierania książek z [Legimi](https://www.legimi.pl/) na czytnik Kindle, napisany w Go.

Program jest całkowicie nieoficjalny i nie jest w żaden sposób powiązany z Legimi.
Korzysta z binarnego protokołu oficjalnej aplikacji Legimi dla Windows (wersja 1.8.5),
odtworzonego na podstawie ruchu sieciowego, dlatego obsługuje tylko część funkcji.

To repozytorium jest forkiem [tp86/legimi-go](https://github.com/tp86/legimi-go)
(który z kolei zastąpił [wcześniejszą wersję w Lua](https://github.com/tp86/legimi/)).
Zobacz [Historia](#historia).

## Instalacja

Zbuduj program ze źródeł (wymagane Go 1.22 lub nowsze):

```shell
$ git clone https://github.com/quiris11/legimi-go.git
$ cd legimi-go
$ go build -o legimi-go .
```

Bez zainstalowanego Go można zbudować go w kontenerze, np. przez Podmana:

```shell
$ podman run --rm -v "$PWD":/src:Z -w /src -e CGO_ENABLED=0 docker.io/library/golang:1.22 go build -o legimi-go .
```

Skopiuj `legimi-go` do katalogu w `PATH` (np. `~/.local/bin`), żeby uruchamiać go z dowolnego miejsca.
Po aktualizacji kodu zbuduj program ponownie (i skopiuj go jeszcze raz).

Wydania (Releases) oraz `go install github.com/tp86/legimi-go@<wersja>` dają oryginalną wersję, bez zmian z tego forka.

## Użycie

```shell
$ legimi-go [opcje] <polecenie> [argumenty]
$ legimi-go --help
```

Polecenie jest obowiązkowe, nie ma polecenia domyślnego.

### Typowe użycie

1.  Podłącz Kindle. Żeby książki trafiały od razu na czytnik, ustaw jego katalog `documents` jako folder docelowy
    (na Fedorze Kindle jest zwykle zamontowany w `/run/media/$USER/Kindle`). Wystarczy zrobić to raz:

    ```shell
    $ legimi-go dir /run/media/$USER/Kindle/documents
    ```

    Zobacz [Folder docelowy](#folder-docelowy).

2.  Wyświetl książki z półki:

    ```shell
    $ legimi-go list
    ```

    Przy pierwszym uruchomieniu program zapyta o login, hasło i numer seryjny Kindle, zobacz [Pierwsze uruchomienie](#pierwsze-uruchomienie).

3.  Pobierz książki, wybierając je z interaktywnej listy:

    ```shell
    $ legimi-go select
    ```

    albo podając ich numery id (pierwsza liczba w wyniku `list`):

    ```shell
    $ legimi-go download <id> [<id> ...]
    ```

    Każda książka zapisuje się jako `<id>.mobi` w folderze docelowym.

4.  Bezpiecznie odłącz Kindle. Pobrane książki pojawią się w jego bibliotece.

### Pierwsze uruchomienie

Gdy brakuje danych logowania albo id czytnika, program o nie pyta:

-   **Login** i **hasło do Legimi** (hasło nie jest wyświetlane podczas wpisywania).
    Jeśli nie podano ich opcjami `--login` / `--password`, program zapisuje je w [pliku konfiguracji](#pliki),
    ale dopiero gdy Legimi je przyjmie (udane logowanie albo rejestracja Kindle).
    Przerwane uruchomienie albo błędne hasło niczego nie zapisuje, a przy następnym uruchomieniu program po prostu zapyta ponownie.
-   **Numer seryjny Kindle** (na czytniku: Ustawienia → Opcje urządzenia → Informacje o urządzeniu), bez spacji.
    Program rejestruje Kindle w Legimi i otrzymuje jego id.
    Zarówno id, jak i numer seryjny zapisuje w pliku konfiguracji.
    Ten sam numer seryjny daje to samo id. Legimi ogranicza liczbę czytników Kindle w abonamencie.
    Błędny numer seryjny może nie dać błędu, tylko id, które nie działa (pusta lista książek).

W jednym uruchomieniu program pyta o dane logowania najwyżej raz.

## Polecenia

### `list`

Wyświetla książki z półki Legimi dostępne na Kindle w dwóch sekcjach: `Downloaded` (pobrane) i `Not downloaded` (niepobrane),
według informacji z Legimi. Przy każdej sekcji podana jest liczba książek. Książki są posortowane według autora, a potem tytułu
(polska kolejność alfabetyczna, bez rozróżniania wielkości liter). Każda linia zawiera id, autora i tytuł:

```
Downloaded (12):
 1000001: Bolesław Prus - "Lalka" [downloaded with legimi-go 2026-01-15 18:30]
  ...
Not downloaded (34):
 1000002: Eliza Orzeszkowa - "Nad Niemnem"
  ...
Downloads left: 7 of 10
```

-   `[downloaded with legimi-go <data>]` oznacza ostatnie udane pobranie tym programem.
-   `[hidden by Legimi after download request]` oznacza książkę, której Legimi przestało pokazywać po prośbie o pobranie,
    zobacz [Książki ukryte przez Legimi](#książki-ukryte-przez-legimi). Pod listą pojawia się wtedy liczba takich książek z podpowiedzią, żeby użyć `refresh`.
-   `Downloads left: N of M` to liczba pobrań pozostałych w bieżącym okresie abonamentu.
    Gdy Legimi jej nie przysyła (np. pakiet nie jest aktywny), wyświetla się `unknown (is your Legimi package active?)`.
-   Przy pustej półce wyświetla się `No books on shelf.`

Dane książek z listy (id, wersja, tytuł, autor, status pobrania) program zapisuje w [pliku książek](#pliki).

### `select`

Interaktywna lista do wybierania książek bez przepisywania numerów id. Wymaga interaktywnego terminala.

Na górze są książki jeszcze niepobrane, niżej pobrane (przyciemnione, z dopiskiem `(downloaded)` albo datą pobrania).
Pasek stanu pokazuje liczbę zaznaczonych książek, ile z nich nie było jeszcze pobranych i ile zostało pobrań.
Gdy zaznaczonych niepobranych książek jest więcej niż pozostałych pobrań, pojawia się ostrzeżenie `NOT ENOUGH DOWNLOADS LEFT`.

| Klawisz | Działanie |
|---|---|
| litery | filtrowanie po autorze i tytule (bez znaczenia wielkość liter i polskie znaki, np. `lukasz` znajdzie `Łukasz`) |
| Backspace / Ctrl+U | usunięcie ostatniego znaku / wyczyszczenie filtra |
| ↑ ↓, Page Up / Page Down, Home / End | ruch po liście |
| Spacja (lub Tab) | zaznaczenie / odznaczenie książki pod kursorem |
| Enter | pobranie zaznaczonych książek (albo książki pod kursorem, jeśli nic nie jest zaznaczone) |
| `y` (lub `t`) | potwierdzenie pobrania; każdy inny klawisz wraca do listy |
| Esc / Ctrl+C | wyjście bez pobierania |

Zaznaczenia nie znikają przy zmianie filtra. Po potwierdzeniu książki są pobierane tak samo jak poleceniem `download`.

### `download <id> ...`

Pobiera książki o podanych numerach id do [folderu docelowego](#folder-docelowy), po kolei.
Jeśli folder nie istnieje (np. Kindle nie jest podłączony), program kończy się błędem, zanim cokolwiek pobierze
(przy `select` jeszcze przed wyświetleniem listy). Dla każdej książki:

1.  Dane książki są pobierane z listy półki w Legimi. Jeśli Legimi już jej nie pokazuje
    (zobacz [Książki ukryte przez Legimi](#książki-ukryte-przez-legimi)), tytuł i autor są brane z [pliku książek](#pliki),
    a aktualna wersja jest ustalana przez pytanie Legimi o kolejne numery wersji.
    Dzięki temu zawsze pobierane jest aktualne wydanie.
2.  Program prosi Legimi o dane pobierania (adres i rozmiar pliku). Gdy Legimi wciąż przygotowuje plik
    (`Waiting for book ... to be ready for download.`), program ponawia prośbę z rosnącymi odstępami przez około 3 minuty.
3.  Plik jest pobierany w częściach po 80 KiB do `<id>.mobi.part`:
    -   każda część zaczyna się od bajtu, na którym kończą się już pobrane dane,
    -   program sprawdza, czy otrzymał dokładnie tę część, o którą prosił (Legimi odpowiada statusem `200` i samą żądaną częścią),
    -   nieudana część (np. zerwane połączenie) jest ponawiana do 3 razy, od miejsca, w którym kończą się dane,
    -   każde zapytanie ma limit czasu 30 sekund.
4.  Plik jest zapisywany fizycznie na dysk i weryfikowany: dokładny rozmiar podany przez Legimi, nagłówek Mobipocket (`BOOKMOBI`),
    spójny spis rekordów i znacznik końca pliku.
5.  Dopiero wtedy dostaje nazwę `<id>.mobi`, zastępując ewentualny istniejący plik. Jeśli coś się nie uda, plik tymczasowy jest usuwany,
    wyświetla się `failed`, a istniejący `<id>.mobi` pozostaje nietknięty.

Postęp wygląda tak: `Downloading book <id>: "<tytuł>" ....... done` (kropka za każdą pobraną część).
Przy pobieraniu wielu książek błąd jednej nie przerywa pozostałych; błędy są wypisywane na końcu.

Prośba o pobranie i udane pobranie są zapisywane w [pliku książek](#pliki).
Legimi wlicza do limitu abonamentu pobrania książek wcześniej niepobranych;
w praktyce ponowne pobranie już pobranej książki nie zmniejszało limitu.

### `dir [folder]`

Pokazuje albo ustawia [folder docelowy](#folder-docelowy) pobieranych książek. Nie łączy się z Legimi.

-   `legimi-go dir` pokazuje aktualne ustawienie (i folder z opcji `--dir`, jeśli ją podano).
-   `legimi-go dir <folder>` zapisuje folder w pliku konfiguracji jako `downloadDir`. Ścieżka jest zapisywana jako bezwzględna
    (`~` oznacza katalog domowy, ścieżka względna jest liczona od bieżącego katalogu).
    Gdy folder w tej chwili nie istnieje (np. Kindle nie jest podłączony), ustawienie i tak jest zapisywane, z ostrzeżeniem.
-   `legimi-go dir .` usuwa ustawienie, czyli książki znowu trafiają do bieżącego katalogu.

Ścieżkę ze spacjami trzeba ująć w cudzysłów.

### `refresh`

Ponownie rejestruje Kindle, używając numeru seryjnego zapisanego w pliku konfiguracji (jeśli go brakuje, program o niego zapyta).
To resetuje stan urządzenia w Legimi, więc książki ukryte przez Legimi znowu pojawiają się na liście.
Id czytnika pozostaje takie samo; jeśli Legimi zwróci inne, program wyświetli ostrzeżenie.

Zaraz po rejestracji Legimi może odpowiedzieć na kolejne zapytanie błędem `INTERNAL_ERROR`; wystarczy uruchomić polecenie ponownie.

### `version`

Wyświetla wersję programu.

## Opcje

Wszystkie opcje są nieobowiązkowe i można je podawać z jednym (`-config`) albo dwoma myślnikami (`--config`).

-   `--config ścieżka`

    Ścieżka do pliku konfiguracji, domyślnie `$HOME/.config/legimi-go/config.ini`.
    Plik książek jest zapisywany obok niego, zobacz [Pliki](#pliki).
    Różne pliki konfiguracji pozwalają przełączać się między kontami Legimi.

-   `--login login`, `--password hasło`

    Dane logowania do Legimi używane zamiast zapisanych w pliku konfiguracji i nigdy do niego niezapisywane.
    Argumenty wiersza poleceń są widoczne dla innych użytkowników komputera (np. w `ps`) i trafiają do historii powłoki,
    więc bezpieczniej jest wpisać hasło, gdy program o nie zapyta.

-   `--dir ścieżka`

    Folder docelowy pobieranych książek, ważniejszy niż `downloadDir` z pliku konfiguracji, zobacz [Folder docelowy](#folder-docelowy).

-   `--debug`

    Wypisuje na stderr wybrane informacje o zapytaniach i odpowiedziach:
    odpowiedź sesji (stan abonamentu, limity urządzeń, pozostałe pobrania) i zapytanie o listę książek.
    Nie wypisuje loginu, hasła ani id sesji.

## Pliki

| Plik | Zawartość |
|---|---|
| `~/.config/legimi-go/config.ini` (albo ścieżka z `--config`) | `login`, `password` (otwartym tekstem), `kindleId`, `kindleSerialNumber`; `downloadDir` (ustawiany poleceniem `dir`) |
| `config-books.json` obok pliku konfiguracji (nazwa tworzona od nazwy pliku konfiguracji, np. `praca.ini` → `praca-books.json`) | dane książek widzianych na półce (wersja, tytuł, autor, status pobrania) oraz daty próśb o pobranie i pobrań tym programem |
| `<id>.mobi` w folderze docelowym | pobrana książka |
| `<id>.mobi.part` w folderze docelowym | książka w trakcie pobierania, usuwana przy błędzie (zostaje tylko po zabiciu programu) |

Katalog konfiguracji jest tworzony z uprawnieniami `700`, a pliki konfiguracji i książek z `600`
(dostęp tylko dla właściciela). Przy starcie program poprawia uprawnienia pliku konfiguracji utworzonego przez starsze wersje.

Hasło, którego format INI nie potrafi odczytać bez zmian (np. otoczone cudzysłowami), nie jest zapisywane.
Program wyświetla ostrzeżenie, a hasło trzeba wpisywać przy każdym uruchomieniu albo podać opcją `--password`.

## Folder docelowy

Pobrane książki trafiają do pierwszego ustawionego z:

1.  opcji `--dir` (np. `legimi-go --dir ~/Książki download 123`),
2.  ustawienia `downloadDir` w pliku konfiguracji, zapisywanego poleceniem [`dir`](#dir-folder), np.:

    ```shell
    $ legimi-go dir ~/Dropbox/Legimi
    ```

    (można je też wpisać ręcznie do pliku konfiguracji jako `downloadDir = ścieżka`),

3.  bieżącego katalogu.

Znak `~` na początku ścieżki oznacza katalog domowy (także w pliku konfiguracji). Folder musi już istnieć,
program go nie tworzy - dzięki temu niepodłączony Kindle daje błąd zamiast pobierania na dysk komputera.

## Książki ukryte przez Legimi

Legimi przestaje pokazywać książkę na liście dla danego urządzenia, gdy tylko poprosi ono o jej pobranie, nawet jeśli pobieranie się nie uda.
Książka zostaje na półce na stronie Legimi, a limit pobrań się nie zmienia.
Program radzi sobie z tym tak:

-   książki ukryte po prośbie o pobranie wysłanej przez ten program są znowu pokazywane przez `list` i `select`
    (na podstawie pliku książek), z adnotacją `[hidden by Legimi after download request]`, i można je pobrać ponownie,
-   `refresh` sprawia, że Legimi znowu je pokazuje, przez ponowną rejestrację Kindle
    (świeża konfiguracja na innym komputerze działa tak samo, bo przy pierwszym uruchomieniu rejestruje Kindle).

Książki usunięte z półki przez Ciebie albo przez Legimi (np. po wygaśnięciu licencji) nie są pokazywane,
bo program nigdy nie prosił o ich pobranie.

## Ograniczenia

-   Obsługiwana jest tylko część funkcji oficjalnej aplikacji, na podstawie odtworzonego protokołu.
-   Program rozpoznaje tylko kilka kodów błędów Legimi (błędne dane logowania, błędne id Kindle, niedostępna wersja książki);
    pozostałe wyświetla jako `error response received: <kod>`.
-   Rejestracja Kindle tym programem działa, ale nie była pierwotnym celem narzędzia; punktem odniesienia jest oficjalna aplikacja.
-   Legimi blokuje pobrania ponad limit abonamentu.
-   Treść książek jest zaszyfrowana (DRM), a Legimi nie przysyła sumy kontrolnej, więc weryfikacja sprawdza rozmiar i strukturę pliku,
    a nie jego treść.

## Rozwiązywanie problemów

-   Pusta lista: sprawdź, czy pakiet Legimi jest aktywny (znana wartość `Downloads left`) i czy numer seryjny Kindle jest poprawny.
    Innym użytkownikom pomogło uruchomienie oficjalnej aplikacji Legimi albo świeża konfiguracja (`--config` z nową ścieżką)
    ([tp86/legimi-go#3](https://github.com/tp86/legimi-go/issues/3)).
-   `book ... is still being prepared by Legimi`: uruchom to samo polecenie `download` później.
-   Szczegóły sesji pokaże opcja `--debug`.

## Rozwój

```shell
$ go test ./...
$ go vet ./...
```

Testy obejmują scenariusze pobierania z lokalnym serwerem HTTP, który symuluje zerwane połączenia i zachowanie serwera Legimi.

## Historia

Oficjalna aplikacja Legimi nie działa na Linuksie.
Autor oryginału chciał pobierać książki z Linuksa bez przełączania się na inny system.
[Pierwsza wersja](https://github.com/tp86/legimi/) powstała w Lua, a potem została przepisana na Go,
żeby łatwiej ją instalować i utrzymywać. Logika została odtworzona na podstawie ruchu
między oficjalną aplikacją Legimi a serwerem.
