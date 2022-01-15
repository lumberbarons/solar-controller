#!/usr/bin/env python3

import sys
import modbus_tk
import modbus_tk.defines as cst
from modbus_tk import modbus_rtu
import serial
import datetime

def main():
    port = sys.argv[1]
    server = modbus_rtu.RtuServer(serial.Serial(port))

    try:
        slave = server.add_slave(1)
        slave.add_block('real_time_data', cst.ANALOG_INPUTS, 0x3100, 0xFF) # real time data
        slave.add_block('real_time_status', cst.ANALOG_INPUTS, 0x3200, 0xFF) # real time status
        slave.add_block('statistics', cst.ANALOG_INPUTS, 0x3300, 0xFF) # statistics

        slave.set_values('real_time_data', 0x3100, [7856, 58]) # array voltage and current
        slave.set_values('real_time_data', 0x3102, [0xCBFF, 0x003]) # array power
        slave.set_values('real_time_data', 0x3104, 2425) # battery voltage

        slave.set_values('real_time_data', 0x3106, [0xCB00, 0x003]) # charging power

        slave.set_values('real_time_data', 0x3110, [65536 - 240, 360]) # temperatures
        slave.set_values('real_time_data', 0x311A, 58) # battery soc

        slave.set_values('real_time_status', 0x3201, 0x08) # controller status

        slave.set_values('statistics', 0x3302, [2801, 2301]) # min max daily voltage

        slave.set_values('statistics', 0x330C, [0x000F, 0x0001]) # day power gen
        slave.set_values('statistics', 0x330E, [0x00FF, 0x0010]) # month power gen
        slave.set_values('statistics', 0x3310, [0x0FFF, 0x0100]) # year power gen
        slave.set_values('statistics', 0x3312, [0xFFFF, 0x1000]) # forever power gen

        slave.add_block('holding_registers', cst.HOLDING_REGISTERS, 0x9000, 0xFF)

        slave.set_values('holding_registers', 0x9000, [3, 445]) # battery type and capacity

        slave.set_values('holding_registers', 0x9006, [2700, 2850, 2900, 2725]) # various voltages

        now = datetime.datetime.now()

        minSec = (now.minute << 8) | now.second
        dayHour = (now.day << 8) | now.hour
        yearMonth = ((now.year - 2000) << 8) | now.month

        slave.set_values('holding_registers', 0x9013, [minSec, dayHour, yearMonth])

        server.start()

        while True:
            cmd = sys.stdin.readline()
            args = cmd.split(' ')
            if cmd.find('quit') == 0:
                break
    finally:
        server.stop()

if __name__ == "__main__":
    main()
