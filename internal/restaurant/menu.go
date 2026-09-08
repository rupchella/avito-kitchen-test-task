package restaurant

func DemoMenu() []MenuItemInput {
	return []MenuItemInput{
		{
			ExternalID:   "pizza-margherita",
			Name:         "Пицца Маргарита",
			Description:  "Томаты, моцарелла, базилик",
			PriceKopecks: 49900,
			IsAvailable:  true,
		},
		{
			ExternalID:   "pizza-pepperoni",
			Name:         "Пицца Пепперони",
			Description:  "Острая салями, моцарелла, томатный соус",
			PriceKopecks: 59900,
			IsAvailable:  true,
		},
		{
			ExternalID:   "pasta-carbonara",
			Name:         "Паста Карбонара",
			Description:  "Спагетти, бекон, сливочный соус, пармезан",
			PriceKopecks: 45000,
			IsAvailable:  true,
		},
		{
			ExternalID:   "salad-caesar",
			Name:         "Салат Цезарь",
			Description:  "Курица, романо, пармезан, соус цезарь",
			PriceKopecks: 39000,
			IsAvailable:  true,
		},
		{
			ExternalID:   "dessert-tiramisu",
			Name:         "Тирамису",
			Description:  "Классический десерт с маскарпоне",
			PriceKopecks: 29000,
			IsAvailable:  false,
		},
		{
			ExternalID:   "drink-lemonade",
			Name:         "Домашний лимонад",
			Description:  "Лимон, мята, лёд",
			PriceKopecks: 19000,
			IsAvailable:  true,
		},
	}
}
