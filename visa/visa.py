# from AppKit import NSWorkspace
from random import random, randrange
from time import sleep, localtime
from playsound import playsound
from osascript import osascript

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.action_chains import ActionChains
from webdriver_manager.chrome import ChromeDriverManager

espera = 60 * 10

email = '' # TODO: pasar por args
password = '' # TODO: pasar por args
driver_path = '/Users/faustofusse/Documents/Software/Random/drivers/chromedriver'

def hour():
    return localtime().tm_hour
def alarma():
    osascript('set volume output volume 100')
    while True: playsound('/Users/faustofusse/Documents/Software/Python/alarm.mp3')

def initialize_driver():
    options = Options()
    options.binary_location = '/Applications/Brave Browser.app/Contents/MacOS/Brave Browser'
    driver = webdriver.Chrome(options = options, executable_path = driver_path)
    # driver = webdriver.Chrome(options = options, executable_path = ChromeDriverManager().install())
    return driver

def checkear_turno(driver):
    actions = ActionChains(driver)
    driver.get('https://ais.usvisa-info.com/es-ar/niv/users/sign_in')
    print('Iniciando sesion...')
    actions.move_to_element(driver.find_element_by_id('user_email')).perform()
    driver.find_element_by_id('user_email').send_keys(email)
    driver.find_element_by_id('user_password').send_keys(password)
    driver.find_element_by_class_name('icheckbox').click()
    sleep(1)
    driver.find_element_by_id('new_user').submit()
    sleep(1)
    driver.get('https://ais.usvisa-info.com/es-ar/niv/schedule/35026881/continue')
    driver.execute_script("window.scrollTo(0, 550)")
    sleep(4)
    style = driver.find_element_by_id('consulate_date_time_not_available').get_attribute('style')
    if (style == 'display: block;'):
        sleep(1)
        driver.get('https://ais.usvisa-info.com/es-ar/niv/users/sign_out')
        sleep(1)
        return
    else:
        alarma()

def main():
    while True:
        # previous_app = NSWorkspace.sharedWorkspace().activeApplication()['NSApplicationName']
        print('Checkeando turno...')
        driver = initialize_driver()
        try:
            checkear_turno(driver)
        except:
            print('Error al checkear turno.')
        driver.quit()
        tiempo = 60 * (10 + randrange(-3,4) + random()) if hour() > 7 else 60 * 60
        print('Sleeping', tiempo / 60, 'minutes')
        sleep(tiempo)
        # if (previous_app != 'iTerm 2'): osascript('tell application \"' + previous_app + '\" to activate window')

main()
