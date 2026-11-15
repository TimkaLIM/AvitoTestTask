CREATE TABLE restaurants(
    id SERIAL PRIMARY KEY,
    title VARCHAR(255)
);
CREATE TABLE menu_items(
    id SERIAL PRIMARY KEY,
    restaurant_id INTEGER NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    name VARCHAR(255),
    price INTEGER
);
CREATE TABLE orders(
    id SERIAL PRIMARY KEY,
    restaurant_id INTEGER NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    user_id INTEGER,
    status VARCHAR(100),
    total_price INT
);
CREATE TABLE order_items(
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id INTEGER NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE,
    quantity INTEGER,
    PRIMARY KEY(order_id,menu_item_id)
);
