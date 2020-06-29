create database httplog;
use httplog;

drop table if exists biz_log;
create table biz_log
(
    id          bigint primary key comment '日志记录ID',
    created     datetime default current_timestamp comment '创建时间',
    started     datetime comment '请求时间',
    end         datetime comment '结束时间',
    cost        int comment '费时毫秒',
    ip          varchar(60) comment '当前机器IP',
    hostname    varchar(60) comment '当前机器名称',
    pid         int comment '应用程序PID',
    biz         varchar(60) comment '当前业务名称',
    req_url     varchar(60) comment '请求url',
    req_method  varchar(60) comment '请求方法',
    rsp_body    varchar(60) comment '响应体',
    echo_name    varchar(60) comment '响应体 httplog:"req_json_name"'
) engine = innodb
  default charset = utf8mb4 comment 'biz_log';
