create table if not exists inbox (
    id uuid not null unique,
    address text not null unique,
    created_at numeric,
    created_by text,
    mg_routeid text,
    ttl numeric,
    failed_to_create bool,
    primary key (id)
);

create table if not exists message (
    inbox_id uuid references inbox(id) on delete cascade,
    message_id uuid not null unique,
    received_at numeric,
    mg_id text,
    sender text,
    from_address text,
    subject text,
    body_html text,
    body_plain text,
    ttl numeric,
    primary key (message_id)
);