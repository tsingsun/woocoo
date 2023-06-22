drop table entimport;
create table entimport
(
    id            UUID default generateUUIDv4(),
    string  String comment 'it is string',
    int8 Int8 comment 'int8',
    int16 Int16 comment 'int16',
    int32 Int32 comment 'int32',
    int64 Int64 comment 'int64',
    int int comment 'int',
    date date comment 'date',
    datetime DateTime comment 'datetime',
    saf_decimal SimpleAggregateFunction(anyLast, Nullable(Decimal(10, 6))) comment 'SAF Null Decimal(10,6)',
    saf_string SimpleAggregateFunction(anyLast, Nullable(String)) comment 'SAF Null String',
    created_at    DateTime,
    created_bt INT,
    updated_at SimpleAggregateFunction(anyLast, DateTime),
    updated_by INT
)
    engine = AggregatingMergeTree PARTITION BY toYYYYMM(updated_at)
        ORDER BY (id, string);
