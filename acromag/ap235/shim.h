#ifndef DEVICE_NAME
#include "../apcommon/apcommon.h"
#include "AP235.h"
#endif
APSTATUS GetAPAddress2(int nhandle, struct mapap235** addr);

int Setup_board_corrected_buffer(struct cblk235 *cfg, unsigned long **scattermap);

void Teardown_board_corrected_buffer(struct cblk235 *cfg);

short* MkDataArray(int size);

void* aligned_malloc(int size, int align);

void aligned_free(void *ptr);

void start_waveform(struct cblk235 *cfg);

void stop_waveform(struct cblk235 *cfg);

unsigned long fetch_status(struct cblk235 *cfg);

void refresh_interrupt(struct cblk235 *cfg, unsigned long status);

void do_DMA_transfer(struct cblk235 *cfg, int channel, uint samples, short *p1, short *p2);

void set_DAC_sample_addresses(struct cblk235 *cfg, int channel);
