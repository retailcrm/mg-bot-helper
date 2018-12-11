create table connection
(
  id         serial not null constraint connection_pkey primary key,
  client_id  varchar(70) not null,
  api_key    varchar(100) not null,
  api_url    varchar(255) not null,
  mg_url    varchar(255) not null,
  mg_token    varchar(100) not null,
  commands   jsonb,
  created_at timestamp with time zone,
  updated_at timestamp with time zone,
  active     boolean,
  lang     varchar(2) not null
);

alter table connection
  add constraint connection_key unique (client_id, mg_token);
