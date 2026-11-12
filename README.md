Базовая Структура:
+ Клиент: Заходит -> Видит список ресторанов -> выбирает ресторан -> смотрит меню -> кладёт блюдо в корзину -> оформляет заказ -> ждёт
+ Ресторан: Получает заказ -> меняет статус на "готовится" -> меняет статус на "в пути" -> меняет статус на "доставлено"

База данных:
+ restaurants: id, title
+ menu: id, restaurant_id, name, price
+ orders: id,restaurant_id,user_id,status,total_price
+ order_items: order_id,menu_item,quantity

