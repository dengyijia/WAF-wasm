import json
import glob
import pandas as pd

class Plotter:
  def __init__(self):
    self.PARAMS = ["Jitter", "SocketCount", "RequestedQPS", "Deployed"]
    self.DEFAULT = {"SocketCount": 16, "RequestedQPS": 1000}
    self.PERCENTS = ["50", "75", "90", "99", "99.9"]
    self.data = self.read_data()

  def read_json(self, filename):
    with open(filename) as file:
      result = json.load(file)
    result["Deployed"] = (filename.split("_")[3] == "deployed")
    return result

  def read_data(self):
    filenames = glob.glob("*.json")
    data = {param: [] for param in self.PARAMS + self.PERCENTS}
    for filename in filenames:
      result = self.read_json(filename)

      # record the parameters
      for param in self.PARAMS:
        data[param].append(result[param])

      # record the percentiles
      percentiles = result["DurationHistogram"]["Percentiles"]
      for percentile in percentiles:
        data[str(percentile["Percentile"])].append(percentile["Value"])

    data = pd.DataFrame(data)
    data[self.PERCENTS] = data[self.PERCENTS].astype(float)
    data["RequestedQPS"] = data["RequestedQPS"].astype(int)
    data = data.groupby(self.PARAMS).mean().reset_index()
    return data

  def select_data(self, jitter, param, default, percent):
    data = self.data

    # select data for varying param
    data = data[data["Jitter"] == jitter]
    data = data[data[default] == self.DEFAULT[default]]

    # set index
    data = data.set_index(param).sort_index()

    # find deployed and undeployed
    deployed_data = data.loc[data["Deployed"], percent]
    undeployed_data = data.loc[~data["Deployed"], percent]
    data = pd.concat([deployed_data, undeployed_data], axis=1)
    data.columns = ["deployed", "undeployed"]
    return data
    

  def plot(self, jitter, param, default, percent):
    data = self.select_data(jitter, param, default, percent)
    plot = data.plot()
    fig = plot.get_figure()
    fig.savefig("jitter={}_param={}.png".format(jitter, param))

  def plot_all(self):
    self.plot(True, self.PARAMS[1], self.PARAMS[2], "90")
    self.plot(False, self.PARAMS[1], self.PARAMS[2], "90")
    self.plot(True, self.PARAMS[2], self.PARAMS[1], "90")
    self.plot(False, self.PARAMS[2], self.PARAMS[1], "90")


if __name__ == '__main__':
  p = Plotter()
  p.plot_all()
