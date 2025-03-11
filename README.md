# glowbyte-hw

- [Задания](#задания)
- [Установка Postgres](#установка-postgres)

## Задания

1. Установить где-то у себя Постгрес (на виртуальной машине, например)
2. На пункте 1 научиться делать бекап и рестор. Уметь отвечать на вопросы, а что там происходит.
3. Научиться отвечать на вопросы, что такое вакуум, вакуум фулл, аналайз и что будет, если их не делать?
4. Типы блокировок - какие есть, для чего нужны, где посмотреть?
5. Что такое MVCC, для чего он нужен, и как реализован в Постгресе?
6. Изучить сервисные таблицы (pg_stat_activity, например). Что там лежит? В каких случаях туда надо идти смотреть?

## Установка Postgres

Конфигурация сервисов `postgres` и `pgadmin` описана в `docker-compose.yml`.
Так же есть сервис приложения на Golang для заполнения бд тестовыми данными.

### Запуск

Только `postgres` и `pgadmin`:

```shell
docker compose up
```

С заполнение базы данных:

```shell
docker compose --profile seed up --build
```

После запуска `pgadmin` будет доступен на [localhost:5050](http://localhost:5050)

### Шаблон `.env`

```env
POSTGRES_PASSWORD=
POSTGRES_DB=
PGADMIN_DEFAULT_EMAIL=
PGADMIN_DEFAULT_PASSWORD=
```

Переименуйте и заполните файл `.env.template`.

## Backup & Restore

Документация: [postgresql.org/docs/backup](https://www.postgresql.org/docs/current/backup.html)

### SQL dump

> The idea behind this dump method is to generate a file with SQL commands that, when fed back to the server, will recreate the database in the same state as it was at the time of the dump.

#### Особенности

- Вывод в `stdout`.
- Позволяет создавать резервные копии с любого удаленного хоста, имеющего доступ к базе данных.
- Требует доступ на чтение ко всем таблицам. Полный бекап базы данных обычно требует привилегий супер-пользователя. Частичные бекапы можно делать например с помощью `-n schema` или `-t table`.
- Вывод, в отличие от других методов, совместим с более новыми версиями PostgreSQL.
- Единственный метод, подходящий для переноса баз данных между различными архитектурами (например 32-бит -> 64-бит).
- Внутренняя консистентность: представляет собой снимок базы данных в начале процесса создания дампа.
- Не блокирует большинство операций с базой данных.

#### Backup

```shell
# docker exec -t <container> pg_dumpall -c -U <role> > <filename>.sql

docker exec -t glowbyte-pg pg_dumpall -c -U postgres > dump_`date +%Y-%m-%d"_"%H_%M_%S`.sql
```

#### Restore

```shell
# cat <filename>.sql | docker exec -i <container> psql -U <role>

cat $(ls dump*.sql -rt created | head -n1) | docker exec -i glowbyte-pg psql -U postgres
```

### File System Level Backup

> The backup strategy is to directly copy the files that PostgreSQL uses to store the data in the database. You can use whatever method you prefer for doing file system backups.

#### Особенности

- Сервер базы данных должен быть выключен и при создании бекапа, и при восстановлении.
- Работает только для полного резервного копирования и восстановления всего кластера.
- Размер файла обычно больше, чем у SQL dump, но процесс может быть быстрее.

#### Backup

Пример из [документации](https://www.postgresql.org/docs/current/backup-file.html):

```shell
tar -cf backup.tar /usr/local/pgsql/data
```

Поскольку в этом проекте используется Docker, volume `/var/lib/postgresql/data` и так прокинут на хост в директорию `./docker/pgdata`. Так что можно работать с ним:

```shell
tar -cf backup.tar ./docker/pgdata
```

#### Restore

```shell
tar -xzf backup.tar -C .
```

### Continuous Archiving and Point-in-Time Recovery (PITR)

> At all times, PostgreSQL maintains a write ahead log (WAL) in the pg_wal/ subdirectory of the cluster's data directory. The log records every change made to the database's data files. This log exists primarily for crash-safety purposes: if the system crashes, the database can be restored to consistency by “replaying” the log entries made since the last checkpoint. However, the existence of the log makes it possible to use a third strategy for backing up databases: we can combine a file-system-level backup with backup of the WAL files. If recovery is needed, we restore the file system backup and then replay from the backed-up WAL files to bring the system to a current state.

#### Особенности

- Нет нужды в идеально консистентном файловом бекапе в качестве начальной точки. Все внутренние несогласованности будут исправлены при проигрывании лога (процесс не принципиально отличается от аварийного восстановления). Достаточно `tar` или чего-то вроде.
- Поскольку мы можем комбинировать бесконечно длинную последовательность файлов WAL для воспроизведения, можно обеспечить непрерывное резервное копирование, просто продолжая архивировать файлы WAL. Это особенно ценно для больших баз данных, где может быть неудобно часто выполнять полное резервное копирование.
- Нет необходимости воспроизводить записи WAL до конца. Мы можем остановить воспроизведение в любой момент и получить согласованный снимок базы данных в том виде, в каком она была на тот момент. Таким образом, этот метод поддерживает восстановление в определенный момент времени: можно восстановить базу данных до ее состояния в любой момент с момента создания базовой резервной копии.
- Если мы постоянно передаем серию файлов WAL на другую машину, на которую был загружен тот же базовый файл резервной копии, у нас есть система "теплого ожидания": в любой момент мы можем запустить вторую машину, и у нее будет почти текущая копия базы данных.
- Работает только для полного резервного копирования и восстановления всего кластера.
- Большие размеры файлов

Заметка из документации:

> pg_dump and pg_dumpall do not produce file-system-level backups and cannot be used as part of a continuous-archiving solution. Such dumps are logical and do not contain enough information to be used by WAL replay.
