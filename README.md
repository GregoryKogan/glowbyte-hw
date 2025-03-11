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
POSTGRES_USER=
POSTGRES_DB=
PGADMIN_DEFAULT_EMAIL=
PGADMIN_DEFAULT_PASSWORD=
```

Переименуйте и заполните файл `.env.template`.
