# Communautofinder Telegram Bot

## Goal

The goal of this Telegram bot is to provide an interface for launching a Communauto car search on Telegram. When it finds at least one car, the bot will return the number of cars found. Interaction with the bot is in french.

## Commands

- **/aide**: View available commands.
- **/chercher**: Begin a discussion with the bot to set search parameters.
- **/recommencer**: Start a new search with the same parameters as the previous one.

## Usage

### Using the Bot

To use this bot, you need to define the following system environment variables:

1. **Telegram Token**:
    - Variable Name: `TOKEN_COMMUNAUTOSEARCH_BOT`
    - Description: This is the token value for Telegram.
    - Example: `TOKEN_COMMUNAUTOSEARCH_BOT=your_telegram_token_value`

2. **Authorized Users**:
    - Variable Name: `AUTHORIZED_USERS_ID`
    - Description: This is a list of user IDs that you want to authorize to chat with the bot. The IDs should be separated by a semicolon (`;`).
    - Example: `AUTHORIZED_USERS_ID=id1;id2;id3`

3. **City ID**:
    - Variable Name: `CITY_ID`
    - Description: This is the ID of the city you want to query from. You can obtain the list of city IDs from this [link](https://restapifrontoffice.reservauto.net/ReservautoFrontOffice/index.html?urls.primaryName=Branch%20version%202%20(6.93.1)#/).
    - Example: `CITY_ID=your_city_id`

Please replace `your_telegram_token_value`, `id1;id2;id3`, and `your_city_id` with your actual values.

After that, run `go run main.go` to launch the bot.  
Then, search for the bot on Telegram to start a discussion with it.

## Dependencies

- [communautofinder](https://github.com/mguaylam/communautofinder)
- [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)

## Thank you

Thanks to craftlion for writing all of this. I forked his project to adapt it to my needs.