# Modbus Register Reference - Epever Solar Controller

## Communication Protocol Overview

- **Protocol**: Standard Modbus-RTU
- **Default Device ID**: 1
- **Baud Rate**: 115200
- **Data Bits**: 8
- **Stop Bits**: 1
- **Flow Control**: None
- **Address Format**: Hexadecimal with base offset 0x00
- **32-bit Data**: Uses two 16-bit registers (L and H)

## Modbus Register Types

| Function Code | Register Type | Access | Description |
|---------------|---------------|--------|-------------|
| 02 | Discrete Inputs | Read-only | Single bit status values (coil status) |
| 04 | Input Registers | Read-only | 16-bit read-only data registers |
| 03 | Holding Registers | Read/Write | 16-bit read/write configuration registers |
| 05 | Coils | Write | Single bit write commands |
| 10 | Holding Registers | Write Multiple | Write multiple 16-bit registers |

---

## A. Real Time Data

Real-time data, status, and historical statistics of energy generated and consumed.

| Number | Variable Name | Address | Function Code | Register Type | Description | Unit | Multiplier |
|--------|---------------|---------|---------------|---------------|-------------|------|------------|
| A1 | Over temperature inside the device | 2000 | 02 (read) | Discrete Input | Temperature inside controller higher than protection point. 1=Over temp, 0=Normal | - | - |
| A2 | Day/Night | 200C | 02 (read) | Discrete Input | 1=Night, 0=Day | - | - |
| A3 | PV array input voltage | 3100 | 04 (read) | Input Register | Solar charge controller PV array voltage | V | 100 |
| A4 | PV array input current | 3101 | 04 (read) | Input Register | Solar charge controller PV array current | A | 100 |
| A5 | PV array input power L | 3102 | 04 (read) | Input Register | Solar charge controller PV array power (low 16 bits) | W | 100 |
| A6 | PV array input power H | 3103 | 04 (read) | Input Register | Solar charge controller PV array power (high 16 bits) | W | 100 |
| A7 | Load voltage | 310C | 04 (read) | Input Register | Load voltage | V | 100 |
| A8 | Load current | 310D | 04 (read) | Input Register | Load current | A | 100 |
| A9 | Load power L | 310E | 04 (read) | Input Register | Load power (low 16 bits) | W | 100 |
| A10 | Load power H | 310F | 04 (read) | Input Register | Load power (high 16 bits) | W | 100 |
| A11 | Battery temperature | 3110 | 04 (read) | Input Register | Battery temperature | °C | 100 |
| A12 | Device temperature | 3111 | 04 (read) | Input Register | Device temperature | °C | 100 |
| A13 | Battery SOC | 311A | 04 (read) | Input Register | The percentage of battery's remaining capacity | % | 1 |
| A14 | Battery's real rated voltage | 311D | 04 (read) | Input Register | Current system rated voltage. 1200/2400/3600/4800 = 12V/24V/36V/48V | V | 100 |
| A15 | Battery status | 3200 | 04 (read) | Input Register | See Battery Status Bits below | - | - |
| A16 | Charging equipment status | 3201 | 04 (read) | Input Register | See Charging Equipment Status Bits below | - | - |
| A17 | Discharging equipment status | 3202 | 04 (read) | Input Register | See Discharging Equipment Status Bits below | - | - |
| A18 | Maximum battery voltage today | 3302 | 04 (read) | Input Register | 00:00 Refresh every day | V | 100 |
| A19 | Minimum battery voltage today | 3303 | 04 (read) | Input Register | 00:00 Refresh every day | V | 100 |
| A20 | Consumed energy today L | 3304 | 04 (read) | Input Register | 00:00 Clear every day | KWH | 100 |
| A21 | Consumed energy today H | 3305 | 04 (read) | Input Register | 00:00 Clear every day | KWH | 100 |
| A22 | Consumed energy this month L | 3306 | 04 (read) | Input Register | 00:00 Clear on the first day of month | KWH | 100 |
| A23 | Consumed energy this month H | 3307 | 04 (read) | Input Register | 00:00 Clear on the first day of month | KWH | 100 |
| A24 | Consumed energy this year L | 3308 | 04 (read) | Input Register | 00:00 Clear on 1, Jan | KWH | 100 |
| A25 | Consumed energy this year H | 3309 | 04 (read) | Input Register | 00:00 Clear on 1, Jan | KWH | 100 |
| A26 | Total consumed energy L | 330A | 04 (read) | Input Register | Total consumed energy (low 16 bits) | KWH | 100 |
| A27 | Total consumed energy H | 330B | 04 (read) | Input Register | Total consumed energy (high 16 bits) | KWH | 100 |
| A28 | Generated energy today L | 330C | 04 (read) | Input Register | 00:00 Clear every day | KWH | 100 |
| A29 | Generated energy today H | 330D | 04 (read) | Input Register | 00:00 Clear every day | KWH | 100 |
| A30 | Generated energy this month L | 330E | 04 (read) | Input Register | 00:00 Clear on the first day of month | KWH | 100 |
| A31 | Generated energy this month H | 330F | 04 (read) | Input Register | 00:00 Clear on the first day of month | KWH | 100 |
| A32 | Generated energy this year L | 3310 | 04 (read) | Input Register | 00:00 Clear on 1, Jan | KWH | 100 |
| A33 | Generated energy this year H | 3311 | 04 (read) | Input Register | 00:00 Clear on 1, Jan | KWH | 100 |
| A34 | Total generated energy L | 3312 | 04 (read) | Input Register | Total generated energy (low 16 bits) | KWH | 100 |
| A35 | Total generated energy H | 3313 | 04 (read) | Input Register | Total generated energy (high 16 bits) | KWH | 100 |
| A36 | Battery voltage | 331A | 04 (read) | Input Register | Battery voltage | V | 100 |
| A37 | Battery current L | 331B | 04 (read) | Input Register | Battery current (low 16 bits) | A | 100 |
| A38 | Battery current H | 331C | 04 (read) | Input Register | Battery current (high 16 bits) | A | 100 |

### Battery Status (A15 - Address 3200)

- **D15**: 1=Wrong identification for rated voltage
- **D8**: Battery inner resistance abnormal (1=abnormal, 0=normal)
- **D7-D4**: Temperature status
  - 00H = Normal
  - 01H = Over Temp (Higher than warning settings)
  - 02H = Low Temp (Lower than warning settings)
- **D3-D0**: Voltage status
  - 00H = Normal
  - 01H = Over Voltage
  - 02H = Under Voltage
  - 03H = Over discharge
  - 04H = Fault

### Charging Equipment Status (A16 - Address 3201)

- **D15-D14**: Input voltage status
  - 00H = Normal
  - 01H = No input power connected
  - 02H = Higher input voltage
  - 03H = Input voltage error
- **D13**: Charging MOSFET is short circuit
- **D12**: Charging or Anti-reverse MOSFET is open circuit
- **D11**: Anti-reverse MOSFET is short circuit
- **D10**: Input is over current
- **D9**: The load is over current
- **D8**: The load is short circuit
- **D7**: Load MOSFET is short circuit
- **D6**: Disequilibrium in three circuits
- **D4**: PV input is short circuit
- **D3-D2**: Charging status
  - 00H = No charging
  - 01H = Float
  - 02H = Boost
  - 03H = Equalization
- **D1**: 0=Normal, 1=Fault
- **D0**: 1=Running, 0=Standby

### Discharging Equipment Status (A17 - Address 3202)

- **D15-D14**: Input voltage
  - 00H = Normal
  - 01H = Input voltage low
  - 02H = Input voltage high
  - 03H = No access
- **D13-D12**: Output power
  - 00H = Light load
  - 01H = Moderate
  - 02H = Rated
  - 03H = Overload
- **D11**: Short circuit
- **D10**: Unable to discharge
- **D9**: Unable to stop discharging
- **D8**: Output voltage abnormal
- **D7**: Input over voltage
- **D6**: Short circuit in high voltage side
- **D5**: Boost over voltage
- **D4**: Output over voltage
- **D1**: 0=Normal, 1=Fault
- **D0**: 1=Running, 0=Standby

---

## B. Battery Parameters

After choosing the battery type, set the corresponding parameters.

| Number | Variable Name | Address | Function Code | Register Type | Description | Unit | Multiplier |
|--------|---------------|---------|---------------|---------------|-------------|------|------------|
| B1 | Rated charging current | 3005 | 04 (read) | Input Register | Rated current to battery | A | 100 |
| B2 | Rated load current | 300E | 04 (read) | Input Register | Rated current to load | A | 100 |
| B3 | Battery's real rated voltage | 311D | 04 (read) | Input Register | Current system rated voltage. 1200/2400/3600/4800 = 12V/24V/36V/48V | V | 100 |
| B4 | Battery type | 9000 | 03 (read), 10 (write) | Holding Register | 0000H=User defined, 0001H=Sealed, 0002H=GEL, 0003H=Flooded | - | - |
| B5 | Battery capacity | 9001 | 03 (read), 10 (write) | Holding Register | Rated capacity of the battery | AH | - |
| B6 | Temperature compensation coefficient | 9002 | 03 (read), 10 (write) | Holding Register | Range 0-9 | mV/°C/2V | 100 |
| B7 | Over voltage disconnect voltage | 9003 | 03 (read), 10 (write) | Holding Register | Over voltage disconnect threshold | V | 100 |
| B8 | Charging limit voltage | 9004 | 03 (read), 10 (write) | Holding Register | Charging limit voltage | V | 100 |
| B9 | Over voltage reconnect voltage | 9005 | 03 (read), 10 (write) | Holding Register | Over voltage reconnect threshold | V | 100 |
| B10 | Equalize charging voltage | 9006 | 03 (read), 10 (write) | Holding Register | Equalize charging voltage | V | 100 |
| B11 | Boost charging voltage | 9007 | 03 (read), 10 (write) | Holding Register | Boost charging voltage | V | 100 |
| B12 | Float charging voltage | 9008 | 03 (read), 10 (write) | Holding Register | Float charging voltage | V | 100 |
| B13 | Boost reconnect charging voltage | 9009 | 03 (read), 10 (write) | Holding Register | Boost reconnect charging voltage | V | 100 |
| B14 | Low voltage reconnect voltage | 900A | 03 (read), 10 (write) | Holding Register | Low voltage reconnect threshold | V | 100 |
| B15 | Under voltage warning recover voltage | 900B | 03 (read), 10 (write) | Holding Register | Under voltage warning recovery | V | 100 |
| B16 | Under voltage warning voltage | 900C | 03 (read), 10 (write) | Holding Register | Under voltage warning threshold | V | 100 |
| B17 | Low voltage disconnect voltage | 900D | 03 (read), 10 (write) | Holding Register | Low voltage disconnect threshold | V | 100 |
| B18 | Discharging limit voltage | 900E | 03 (read), 10 (write) | Holding Register | Discharging limit voltage | V | 100 |
| B19 | Battery rated voltage level | 9067 | 03 (read), 10 (write) | Holding Register | 0=auto, 1=12V, 2=24V, 3=36V, 4=48V, 5=60V, 6=110V, 7=120V, 8=220V, 9=240V | - | - |
| B20 | Default load On/Off in manual mode | 906A | 03 (read), 10 (write) | Holding Register | 0=off, 1=on | - | - |
| B21 | Equalize duration | 906B | 03 (read), 10 (write) | Holding Register | Usually 60-120 minutes | Min | - |
| B22 | Boost duration | 906C | 03 (read), 10 (write) | Holding Register | Usually 60-120 minutes | Min | - |
| B23 | Battery discharge | 906D | 03 (read), 10 (write) | Holding Register | Usually 20%-80%. Percentage of battery remaining capacity when stop charging | % | 100 |
| B24 | Battery charge | 906E | 03 (read), 10 (write) | Holding Register | Depth of charge, 100% | % | 100 |
| B25 | Charging mode | 9070 | 03 (read), 10 (write) | Holding Register | Management modes: 0=voltage compensation, 1=SOC | - | - |

### Voltage Parameters Limit Conditions

1. Over voltage disconnect voltage > Charge limit voltage > Equalize charging voltage > Boost charging voltage > Float charging voltage > Boost reconnect charging voltage
2. Under voltage warning recover voltage > Under voltage warning voltage > Low voltage disconnect voltage > Discharging limit voltage
3. Over voltage disconnect voltage > Over voltage reconnect voltage
4. Low voltage reconnect voltage > Low voltage disconnect voltage

---

## C. Load Parameters

Set the load control mode to meet customer's demand.

| Number | Variable Name | Address | Function Code | Register Type | Description | Unit | Multiplier |
|--------|---------------|---------|---------------|---------------|-------------|------|------------|
| C1 | Manual control the load | 2 | 05 (write) | Coil | When load is manual mode: 1=manual on, 0=manual off | - | - |
| C2 | Night time threshold voltage (NTTV) | 901E | 03 (read), 10 (write) | Holding Register | PV voltage lower than this = sundown | V | 100 |
| C3 | Light signal startup (night) delay time | 901F | 03 (read), 10 (write) | Holding Register | PV voltage < NTTV for this duration = night | Min | - |
| C4 | Day time threshold voltage (DTTV) | 9020 | 03 (read), 10 (write) | Holding Register | PV voltage higher than this = sunrise | V | 100 |
| C5 | Light signal close (day) delay time | 9021 | 03 (read), 10 (write) | Holding Register | PV voltage > DTTV for this duration = day | Min | - |
| C6 | Load control mode | 903D | 03 (read), 10 (write) | Holding Register | 0000H=Manual, 0001H=Light ON/OFF, 0002H=Light ON+Timer, 0003H=Timing | - | - |
| C7 | Light on + time (time1) | 903E | 03 (read), 10 (write) | Holding Register | Load output timer1. D15-D8=hour, D7-D0=minute | - | - |
| C8 | Light on + time (time2) | 903F | 03 (read), 10 (write) | Holding Register | Load output timer2. D15-D8=hour, D7-D0=minute | - | - |
| C9 | Timing control (turn on time1) | 9042 | 03 (read), 10 (write) | Holding Register | Turn on/off time of load output | S | - |
| C10 | - | 9043 | 03 (read), 10 (write) | Holding Register | - | Min | - |
| C11 | - | 9044 | 03 (read), 10 (write) | Holding Register | - | H | - |
| C12 | Timing control (turn off time1) | 9045 | 03 (read), 10 (write) | Holding Register | Turn on/off time of load output | S | - |
| C13 | - | 9046 | 03 (read), 10 (write) | Holding Register | - | Min | - |
| C14 | - | 9047 | 03 (read), 10 (write) | Holding Register | - | H | - |
| C15 | Timing control (turn on time2) | 9048 | 03 (read), 10 (write) | Holding Register | Turn on/off time of load output | S | - |
| C16 | - | 9049 | 03 (read), 10 (write) | Holding Register | - | Min | - |
| C17 | - | 904A | 03 (read), 10 (write) | Holding Register | - | H | - |
| C18 | Timing control (turn off time2) | 904B | 03 (read), 10 (write) | Holding Register | Turn on/off time of load output | S | - |
| C19 | - | 904C | 03 (read), 10 (write) | Holding Register | - | Min | - |
| C20 | - | 904D | 03 (read), 10 (write) | Holding Register | - | H | - |
| C21 | Night time | 9065 | 03 (read), 10 (write) | Holding Register | Default whole night length. D15-D8=hour, D7-D0=minute | - | - |
| C22 | Timing control (time choose) | 9069 | 03 (read), 10 (write) | Holding Register | Record time of load. 0=use one time, 1=use two times | - | - |
| C23 | Default load On/Off in manual mode | 906A | 03 (read), 10 (write) | Holding Register | 0=off, 1=on | - | - |

---

## D. Real Time Clock

| Number | Variable Name | Address | Function Code | Register Type | Description | Unit | Multiplier |
|--------|---------------|---------|---------------|---------------|-------------|------|------------|
| D1 | Real time clock | 9013 | 03 (read), 10 (write) | Holding Register | D7-0=Sec, D15-8=Min. Must be written simultaneously | - | - |
| D2 | Real time clock | 9014 | 03 (read), 10 (write) | Holding Register | D7-0=Hour, D15-8=Day | - | - |
| D3 | Real time clock | 9015 | 03 (read), 10 (write) | Holding Register | D7-0=Month, D15-8=Year | - | - |

---

## E. Device Parameters

| Number | Variable Name | Address | Function Code | Register Type | Description | Unit | Multiplier |
|--------|---------------|---------|---------------|---------------|-------------|------|------------|
| E1 | Battery upper temperature limit | 9017 | 03 (read), 10 (write) | Holding Register | Battery upper temperature limit | °C | 100 |
| E2 | Battery lower temperature limit | 9018 | 03 (read), 10 (write) | Holding Register | Battery lower temperature limit | °C | 100 |
| E3 | Device over temperature | 9019 | 03 (read), 10 (write) | Holding Register | Device over temperature threshold | °C | 100 |
| E4 | Device recovery temperature | 901A | 03 (read), 10 (write) | Holding Register | Device recovery temperature | °C | 100 |
| E5 | Backlight time | 9063 | 03 (read), 10 (write) | Holding Register | Close after LCD backlight setting (seconds) | S | - |

---

## F. Rated Parameters

| Number | Variable Name | Address | Function Code | Register Type | Description | Unit | Multiplier |
|--------|---------------|---------|---------------|---------------|-------------|------|------------|
| F1 | Array rated voltage | 3000 | 04 (read) | Input Register | PV array rated voltage | V | 100 |
| F2 | Array rated current | 3001 | 04 (read) | Input Register | PV array rated current | A | 100 |
| F3 | Array rated power L | 3002 | 04 (read) | Input Register | PV array rated power (low 16 bits) | W | 100 |
| F4 | Array rated power H | 3003 | 04 (read) | Input Register | PV array rated power (high 16 bits) | W | 100 |
| F5 | Battery rated voltage | 3004 | 04 (read) | Input Register | Rated voltage to battery | V | 100 |
| F6 | Battery rated current | 3005 | 04 (read) | Input Register | Rated current to battery | A | 100 |
| F7 | Battery rated power L | 3006 | 04 (read) | Input Register | Rated power to battery (low 16 bits) | W | 100 |
| F8 | Battery rated power H | 3007 | 04 (read) | Input Register | Rated power to battery (high 16 bits) | W | 100 |
| F9 | Rated load voltage | 300D | 04 (read) | Input Register | Rated voltage to load | V | 100 |
| F10 | Rated load current | 300E | 04 (read) | Input Register | Rated current to load | A | 100 |
| F11 | Rated load power L | 300F | 04 (read) | Input Register | Rated power to load (low 16 bits) | W | 100 |
| F12 | Rated load power H | 3010 | 04 (read) | Input Register | Rated power to load (high 16 bits) | W | 100 |

---

## G. Other Switching Values

| Number | Variable Name | Address | Function Code | Register Type | Description | Unit | Multiplier |
|--------|---------------|---------|---------------|---------------|-------------|------|------------|
| G1 | Charging device on/off | 0 | 05 (write) | Coil | 1=Charging device on, 0=Charging device off | - | - |
| G2 | Output control mode manual/automatic | 1 | 05 (write) | Coil | 1=Manual, 0=Automatic | - | - |
| G3 | Manual control the load | 2 | 05 (write) | Coil | When load is manual mode: 1=on, 0=off | - | - |
| G4 | Default control the load | 3 | 05 (write) | Coil | When load is default mode: 1=on, 0=off | - | - |
| G5 | Enable load test mode | 5 | 05 (write) | Coil | 1=Enable, 0=Disable (normal) | - | - |
| G6 | Force the load on/off | 6 | 05 (write) | Coil | 1=Turn on, 0=Turn off (for temporary test) | - | - |
| G7 | Restore system defaults | 13 | 05 (write) | Coil | 1=yes, 0=no | - | - |
| G8 | Clear generating electricity statistics | 14 | 05 (write) | Coil | 1=clear (root privileges required) | - | - |

---

## Status Analysis Reference

- **Array status**: Address 3201 bits D15-D10
- **Charging status**: Address 3201 bits D3-D2
- **Battery status**: Address 3200 bits D7-D0
- **Load status**: Address 3201 bits D9-D7, Address 3202 bits D13-D8, D6-D4
- **Device status**: Address 3200 bit D15, Address 3201 bit D6, Address 2000

---

## Register Type Summary

### Discrete Inputs (Function Code 02)
Read-only single-bit status values. Used for binary sensors and flags.
- **Address Range**: 0x2000-0x200C
- **Examples**: Over temperature flag, Day/Night status

### Input Registers (Function Code 04)
Read-only 16-bit data registers. Used for real-time measurements and status information that cannot be modified.
- **Address Ranges**: 0x3000-0x3010, 0x3100-0x311D, 0x3200-0x3202, 0x3302-0x3313, 0x331A-0x331C
- **Examples**: PV voltage/current/power, battery SOC, temperatures, energy statistics, rated parameters

### Holding Registers (Function Code 03 read, 10 write)
Read/write 16-bit configuration registers. Used for device settings and parameters that can be modified.
- **Address Ranges**: 0x9000-0x902D, 0x903D-0x904D, 0x9063-0x9070
- **Examples**: Battery parameters, voltage thresholds, load control settings, timers, RTC

### Coils (Function Code 05)
Write-only single-bit command values. Used for direct control commands.
- **Address Range**: 0-14
- **Examples**: Manual load control, charging device on/off, system reset commands

### Notes on 32-bit Values
Many registers use paired L (Low) and H (High) 16-bit registers to represent 32-bit values:
- Read both registers sequentially
- Combine as: `value = (H << 16) | L`
- Apply the multiplier/divider to get the actual value
- Example: PV Power = (3103 << 16 | 3102) / 100
