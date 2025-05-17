const getTypeIcon = (type) => `/images/${type.charAt(0).toUpperCase() + type.slice(1)}.png`;

const getExecutionTime = (execution_time) => {
    try {
        const minutes = execution_time.split("m")[0]
        const seconds = execution_time.split("m")[1].split(".")[0]
        if (seconds.length === 1){
            return `${minutes}:0${seconds}`
        }
        return  `${minutes}:${seconds}`
    } catch {
        return "---"
    }
}

const getResultLength = (scan) => {
    try {
        const scanLength = scan.result.split('\n').length - 1
        return scanLength
    } catch {
        return "---"
    }
}

const getLastScanDate = (amassScans) => {
  if (amassScans.length === 0) return 'No scans available';
  const lastScan = amassScans.reduce((latest, scan) => {
    const scanDate = new Date(scan.created_at);
    return scanDate > new Date(latest.created_at) ? scan : latest;
  }, { created_at: '1970-01-01T00:00:00Z' });
  const parsedDate = new Date(lastScan.created_at);
  return isNaN(parsedDate.getTime()) ? 'Invalid scan date' : parsedDate.toLocaleString();
};

const getLatestScanStatus = (amassScans) => {
  if (amassScans.length === 0) return 'No scans available';
  const latestScan = amassScans.reduce((latest, scan) => {
    return new Date(scan.created_at) > new Date(latest.created_at) ? scan : latest;
  }, amassScans[0]);
  return latestScan.status || 'No status available';
};

const getLatestScanTime = (amassScans) => {
  if (amassScans.length === 0) return 'No scans available';
  const latestScan = amassScans.reduce((latest, scan) => {
    return new Date(scan.created_at) > new Date(latest.created_at) ? scan : latest;
  }, amassScans[0]);
  return latestScan.execution_time || '---';
};

const getLatestScanId = (amassScans) => {
  if (amassScans.length === 0) return 'No scans available';
  const latestScan = amassScans.reduce((latest, scan) => {
    return new Date(scan.created_at) > new Date(latest.created_at) ? scan : latest;
  }, amassScans[0]);
  return latestScan.scan_id || 'No scan ID available';
};

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (err) {
    console.error('Failed to copy text: ', err);
    return false;
  }
};

export const getHttpxResultsCount = (scan) => {  console.log("[DEBUG getHttpxResultsCount] Input scan:", scan);  console.log("[DEBUG getHttpxResultsCount] Input scan type:", typeof scan);    if (!scan) {    console.log("[DEBUG getHttpxResultsCount] Scan is null or undefined");    return 0;  }    if (!scan.result) {    console.log("[DEBUG getHttpxResultsCount] Scan has no result property");    return 0;  }    if (!scan.result.String) {    console.log("[DEBUG getHttpxResultsCount] Scan result has no String property");    if (typeof scan.result === 'string') {      console.log("[DEBUG getHttpxResultsCount] Scan result is a string, trying to use it directly");      const count = scan.result.split('\n').filter(line => line.trim()).length;      console.log("[DEBUG getHttpxResultsCount] Direct string count:", count);      return count;    }    return 0;  }    const count = scan.result.String.split('\n').filter(line => line.trim()).length;  console.log("[DEBUG getHttpxResultsCount] Final count:", count);  return count;};

export { getTypeIcon, getLastScanDate, getLatestScanStatus, getLatestScanTime, getLatestScanId, getExecutionTime, getResultLength, copyToClipboard };
