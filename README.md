# trader

## pkg/connectors/quikservice
Данный пакет позволяет работать с Quik аналогично проекту [QuikSharp](https://github.com/finsight/QUIKSharp).
Какие отличия от QuikSharp:
- тк обращение к QuikService все равно спользует mutex, то не очевидна польза от асинхронного API.
- затруднительно доказать корректность [QuikService.cs](https://github.com/finsight/QUIKSharp/blob/master/src/QuikSharp/QuikService.cs) (конечно использовать решения c закрытым кодом - вообще не вариант).
- Кажется логичным все json ответы десериализовать в статичиские типизированныые структуры, но если большинство полей часто не используются, а сами поля могут меняться, то такое API может быть более хрупким, чем получение результата в виде нетипизированного словаря.

## pkg/brokers
Если захотим использовать разные коннекторы для разных брокеров, то хочется, чтобы торговые системы не зависели от конкретных коннекторов, а иметь общее API.

## pkg/strategies
Позволяет автоматически торговать советников, если советник возвращает прогноз в отрезке [-1, +1].

## Ссылки
+ [Авторизация в Quik](https://github.com/finsight/QUIKSharp/tree/master/Examples/AutoConnector)
+ [Lua скрипты для Quik](https://github.com/finsight/QUIKSharp/tree/master/src/QuikSharp/lua)
