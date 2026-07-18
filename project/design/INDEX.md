# agentrepl — Design Index
The manifest for the split design. Each Decision lives in its own `project/design/DNN.md` (zero-padded filename; referenced in prose and the plan as `D<N>`, e.g. `D5`). `project/design/README.md` holds only the invariant spine — Authority, the *Requirement ids* convention, and *Conventions*. This index maps every Decision **and** every `R-XXXX-XXXX` Verification id to its file, so the build loop (and any reader) jumps straight to the one Decision a phase realizes without loading the whole design. Resolve a Decision by its number below; resolve an id either by grepping this index (`grep -n R-4IL7-JPW0 project/design/INDEX.md`) or the files directly (`grep -rl R-4IL7-JPW0 project/design/`). Append-in-place authority is unchanged: when a Decision changes it is rewritten in its `DNN.md`; this index is regenerated alongside.
## Decisions
One line per Decision, in number order — file, label, and the Verification ids it owns.
- **D1** `project/design/D01.md` — Package layout & seams (the testable skeleton)
  - ids: (none — structural)
- **D2** `project/design/D02.md` — Provider table & agentkit-catalog sourcing
  - ids: R-4IL7-JPW0, R-4JT3-XHMP, R-4L10-B9DE, R-4M8W-P143, R-4NGT-2SUS, R-4OOP-GKLH, R-4PWL-UCC6, R-4R4I-842V
- **D3** `project/design/D03.md` — Config-key namespace, resolution & coercion
  - ids: R-4TKA-ZNK9, R-4W03-R71N, R-4X80-4YSC, R-4YFW-IQJ1, R-4ZNS-WI9Q, R-50VP-AA0F, R-523L-O1R4, R-53BI-1THT, R-54JE-FL8I, R-FZCE-VXJL, R-G304-18RO, R-LYK7-Y7ZS, R-LZS4-BZQH, R-M4NP-V2P9, R-M5VM-8UFY
- **D4** `project/design/D04.md` — CLI flags
  - ids: R-EU69-75V4, R-EWM1-YPCI, R-EXTY-CH37, R-EZ1U-Q8TW
- **D5** `project/design/D05.md` — Turn execution, the Renderer, and color
  - ids: R-CCHP-S6AJ, R-CDPM-5Y18, R-G480-F0ID, R-G5FW-SS92, R-JFBW-TYU8, R-JGJT-7QKX, R-LL9K-SKDQ, R-LNPD-K3V4, R-LOX9-XVLT, R-LRD2-PF37, R-LSKZ-36TW, R-OBNM-N6XX, R-Q52T-PXCR
- **D6** `project/design/D06.md` — REPL lifecycle: exit, interrupt, and log integrity
  - ids: R-LW8O-8I1Z, R-LXGK-M9SO, R-LYOH-01JD, R-M149-RL0R
- **D7** `project/design/D07.md` — Usage & cost reporting
  - ids: R-5J67-0U4U, R-ONJY-6PJG, R-OORU-KHA5, R-OPZQ-Y90U, R-OR7N-C0RJ, R-OSFJ-PSI8, R-OUVC-HBZM, R-OW38-V3QB
- **D8** `project/design/D08.md` — Session log & session-id
  - ids: R-8GF4-LRYU, R-8HN0-ZJPJ, R-8IUX-DBG8, R-8K2T-R36X
- **D9** `project/design/D09.md` — Slash-command dispatch & the command set
  - ids: R-55RA-TCZ7, R-56Z7-74PW, R-BI0J-TIHX, R-BJ8G-7A8M, R-BKGC-L1ZB, R-BLO8-YTQ0, R-BMW5-CLGP, R-BO41-QD7E
- **D10** `project/design/D10.md` — Built-in tools (bash / read / write / edit)
  - ids: R-NHBW-446N, R-NIJS-HVXC, R-NKZL-9FEQ, R-NM7H-N75F, R-NNFE-0YW4, R-NONA-EQMT
- **D11** `project/design/D11.md` — Error handling & REPL resilience
  - ids: R-H7HT-LNRE, R-H8PP-ZFI3, R-H9XM-D78S, R-HB5I-QYZH, R-HCDF-4QQ6
- **D12** `project/design/D12.md` — The self-describing `--help` catalog
  - ids: R-5873-KWGL, R-5AMW-CFXZ, R-5BUS-Q7OO, R-5D2P-3ZFD, R-6DEO-9TXQ, R-6DEO-KEYS, R-AFZE-5WGV, R-APDX-FP3D, R-FT8W-Z2U4, R-FVOP-QMBI, R-FWWM-4E27, R-FY4I-I5SW, R-ODOF-XOTJ, R-OEWC-BGK8
- **D13** `project/design/D13.md` — Wait status line (`waiting for <model>`)
  - ids: R-6DZ8-F5IK, R-6F74-SX99, R-6HMX-KGQN
- **D14** `project/design/D14.md` — Subscription auth, `/login`, and lazy credential failure
  - ids: R-5EAL-HR62, R-5FIH-VIWR, R-5GQE-9ANG, R-5HYA-N2E5

## Verification ids → Decision
Every minted id, sorted, mapped to its Decision and file (grep target for id lookup).
R-4IL7-JPW0  D2  project/design/D02.md
R-4JT3-XHMP  D2  project/design/D02.md
R-4L10-B9DE  D2  project/design/D02.md
R-4M8W-P143  D2  project/design/D02.md
R-4NGT-2SUS  D2  project/design/D02.md
R-4OOP-GKLH  D2  project/design/D02.md
R-4PWL-UCC6  D2  project/design/D02.md
R-4R4I-842V  D2  project/design/D02.md
R-4TKA-ZNK9  D3  project/design/D03.md
R-4W03-R71N  D3  project/design/D03.md
R-4X80-4YSC  D3  project/design/D03.md
R-4YFW-IQJ1  D3  project/design/D03.md
R-4ZNS-WI9Q  D3  project/design/D03.md
R-50VP-AA0F  D3  project/design/D03.md
R-523L-O1R4  D3  project/design/D03.md
R-53BI-1THT  D3  project/design/D03.md
R-54JE-FL8I  D3  project/design/D03.md
R-55RA-TCZ7  D9  project/design/D09.md
R-56Z7-74PW  D9  project/design/D09.md
R-5873-KWGL  D12  project/design/D12.md
R-5AMW-CFXZ  D12  project/design/D12.md
R-5BUS-Q7OO  D12  project/design/D12.md
R-5D2P-3ZFD  D12  project/design/D12.md
R-5EAL-HR62  D14  project/design/D14.md
R-5FIH-VIWR  D14  project/design/D14.md
R-5GQE-9ANG  D14  project/design/D14.md
R-5HYA-N2E5  D14  project/design/D14.md
R-5J67-0U4U  D7  project/design/D07.md
R-6DEO-9TXQ  D12  project/design/D12.md
R-6DEO-KEYS  D12  project/design/D12.md
R-6DZ8-F5IK  D13  project/design/D13.md
R-6F74-SX99  D13  project/design/D13.md
R-6HMX-KGQN  D13  project/design/D13.md
R-8GF4-LRYU  D8  project/design/D08.md
R-8HN0-ZJPJ  D8  project/design/D08.md
R-8IUX-DBG8  D8  project/design/D08.md
R-8K2T-R36X  D8  project/design/D08.md
R-AFZE-5WGV  D12  project/design/D12.md
R-APDX-FP3D  D12  project/design/D12.md
R-BI0J-TIHX  D9  project/design/D09.md
R-BJ8G-7A8M  D9  project/design/D09.md
R-BKGC-L1ZB  D9  project/design/D09.md
R-BLO8-YTQ0  D9  project/design/D09.md
R-BMW5-CLGP  D9  project/design/D09.md
R-BO41-QD7E  D9  project/design/D09.md
R-CCHP-S6AJ  D5  project/design/D05.md
R-CDPM-5Y18  D5  project/design/D05.md
R-EU69-75V4  D4  project/design/D04.md
R-EWM1-YPCI  D4  project/design/D04.md
R-EXTY-CH37  D4  project/design/D04.md
R-EZ1U-Q8TW  D4  project/design/D04.md
R-FT8W-Z2U4  D12  project/design/D12.md
R-FVOP-QMBI  D12  project/design/D12.md
R-FWWM-4E27  D12  project/design/D12.md
R-FY4I-I5SW  D12  project/design/D12.md
R-FZCE-VXJL  D3  project/design/D03.md
R-G304-18RO  D3  project/design/D03.md
R-G480-F0ID  D5  project/design/D05.md
R-G5FW-SS92  D5  project/design/D05.md
R-H7HT-LNRE  D11  project/design/D11.md
R-H8PP-ZFI3  D11  project/design/D11.md
R-H9XM-D78S  D11  project/design/D11.md
R-HB5I-QYZH  D11  project/design/D11.md
R-HCDF-4QQ6  D11  project/design/D11.md
R-JFBW-TYU8  D5  project/design/D05.md
R-JGJT-7QKX  D5  project/design/D05.md
R-LL9K-SKDQ  D5  project/design/D05.md
R-LNPD-K3V4  D5  project/design/D05.md
R-LOX9-XVLT  D5  project/design/D05.md
R-LRD2-PF37  D5  project/design/D05.md
R-LSKZ-36TW  D5  project/design/D05.md
R-LW8O-8I1Z  D6  project/design/D06.md
R-LXGK-M9SO  D6  project/design/D06.md
R-LYK7-Y7ZS  D3  project/design/D03.md
R-LYOH-01JD  D6  project/design/D06.md
R-LZS4-BZQH  D3  project/design/D03.md
R-M149-RL0R  D6  project/design/D06.md
R-M4NP-V2P9  D3  project/design/D03.md
R-M5VM-8UFY  D3  project/design/D03.md
R-NHBW-446N  D10  project/design/D10.md
R-NIJS-HVXC  D10  project/design/D10.md
R-NKZL-9FEQ  D10  project/design/D10.md
R-NM7H-N75F  D10  project/design/D10.md
R-NNFE-0YW4  D10  project/design/D10.md
R-NONA-EQMT  D10  project/design/D10.md
R-OBNM-N6XX  D5  project/design/D05.md
R-ODOF-XOTJ  D12  project/design/D12.md
R-OEWC-BGK8  D12  project/design/D12.md
R-ONJY-6PJG  D7  project/design/D07.md
R-OORU-KHA5  D7  project/design/D07.md
R-OPZQ-Y90U  D7  project/design/D07.md
R-OR7N-C0RJ  D7  project/design/D07.md
R-OSFJ-PSI8  D7  project/design/D07.md
R-OUVC-HBZM  D7  project/design/D07.md
R-OW38-V3QB  D7  project/design/D07.md
R-Q52T-PXCR  D5  project/design/D05.md
