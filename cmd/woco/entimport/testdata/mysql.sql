create database if not exists test;

use test;

drop table if exists entimport;

CREATE TABLE entimport
(
    `id`          int         NOT NULL AUTO_INCREMENT,
    `string`      varchar(45) NOT NULL COMMENT 'it is string',
    `int8`        tinyint     NOT NULL COMMENT 'int8',
    `int16`       smallint    NOT NULL COMMENT 'int16',
    `int32`       mediumint   NOT NULL COMMENT 'int32',
    `int64`       bigint      NOT NULL COMMENT 'int64',
    `int`         int         NOT NULL COMMENT 'int',
    `date`        date        NOT NULL COMMENT 'date',
    `datetime`    datetime    NOT NULL COMMENT 'datetime',
    `saf_decimal` decimal(10, 6)       DEFAULT 0 COMMENT 'SAF Null Decimal(10,6)',
    `saf_string`  varchar(45)          DEFAULT 'saf' COMMENT 'SAF Null String',
    `created_at`  timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_bt`  int         NOT NULL,
    `updated_at`  timestamp   NOT NULL,
    `updated_by`  int         NOT NULL,
    PRIMARY KEY (`id`),
    INDEX `idx_string_int` (`id`, `string`)
) ENGINE=InnoDB;

